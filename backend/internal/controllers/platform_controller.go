package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/cambodia-finance/income-expense/internal/winlose"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlatformController: platforms, HQ-platform assignments, weekly win/lose entry and the win/lose
// reports. An HQ is a branch: everything is limited to the actor's
// own branches (owner/admin see every branch).
type PlatformController struct {
	svc   *services.PlatformService
	db    *gorm.DB
	audit *audit.Service
}

func NewPlatformController(svc *services.PlatformService, db *gorm.DB, auditSvc *audit.Service) *PlatformController {
	return &PlatformController{svc: svc, db: db, audit: auditSvc}
}

func (pc *PlatformController) scope(c *gin.Context) (*accessScope, services.BranchScope, bool) {
	scope, ok := loadScope(c, pc.db)
	if !ok {
		return nil, services.BranchScope{}, false
	}
	return scope, services.BranchScope{All: scope.AllBranches, IDs: scope.BranchIDs}, true
}

// actorOf is the signed-in user (the auth middleware has already checked there is one).
func actorOf(c *gin.Context) uuid.UUID {
	id, _ := getActorID(c)
	return id
}

// ownGroupUnless is the group asked for in the query, else the person's own (the PLATFORMS
// pages don't ask for a group).
func (pc *PlatformController) ownGroupUnless(c *gin.Context, scope services.BranchScope) (*uuid.UUID, bool) {
	branchID, ok := queryID(c, "branch_id")
	if !ok || branchID != nil {
		return branchID, ok
	}
	own, err := pc.svc.OwnGroup(c.Request.Context(), actorOf(c), scope)
	if err != nil {
		writePlatformError(c, err)
		return nil, false
	}
	return &own, true
}

func writePlatformError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrPlatformNotFound), errors.Is(err, services.ErrPartnerNotFound), errors.Is(err, services.ErrAssignmentNotFound),
		errors.Is(err, services.ErrGroupNotFound), errors.Is(err, services.ErrEntryNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrBranchAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only work in your own groups"})
	default:
		writeServiceError(c, err)
	}
}

func (pc *PlatformController) record(c *gin.Context, actor uuid.UUID, action, entityType, entity string, branch *uuid.UUID, details map[string]interface{}) {
	if pc.audit == nil {
		return
	}
	pc.audit.Record(c.Request.Context(), audit.Event{
		ActorID: &actor, Action: action, EntityType: entityType, EntityID: entity, BranchID: branch,
		Outcome: "success", IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), Details: details,
	})
}

// ===== Platforms =====

// listPage reads page and page_size for the lists that page only when asked, so the forms that
// need every row (agent pickers, the group filter) still call them as before. size is 1 to 100
// (20 when missing); on is false when neither was sent.
func listPage(c *gin.Context) (page, size int, on bool) {
	_, hasPage := c.GetQuery("page")
	_, hasSize := c.GetQuery("page_size")
	if !hasPage && !hasSize {
		return 0, 0, false
	}
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	return page, min(size, 100), true
}

// pagedJSON answers a list: the page with its total when it was asked for, else every row.
func pagedJSON(c *gin.Context, items interface{}, total int64, page, size int, on bool) {
	if !on {
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

// ListPlatforms lists the agents the person may see: those of their branches and the shared ones
// (an owner/admin sees every branch's). branch_id: only that branch's agents; assignable=true: the
// agents that can go into the person's own group (its branch's and the shared ones).
func (pc *PlatformController) ListPlatforms(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	page, size, on := listPage(c)
	f := services.PlatformFilter{IncludeInactive: c.Query("include_inactive") == "true", Search: c.Query("search"), Scope: &scope}
	if on {
		f.Page, f.PageSize = page, size
	}
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	if branchID != nil {
		if !scope.Has(*branchID) {
			writePlatformError(c, services.ErrBranchAccessDenied)
			return
		}
		f.BranchID = branchID
	}
	if c.Query("assignable") == "true" {
		// For the branch asked for, else the person's own group.
		if branchID != nil {
			f.WithShared = true
		} else if own, err := pc.svc.OwnGroup(c.Request.Context(), actorOf(c), scope); err == nil {
			f.BranchID, f.WithShared = &own, true
		}
	}
	rows, total, err := pc.svc.ListPlatformsPage(c.Request.Context(), f)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pagedJSON(c, orEmpty(rows), total, page, size, on)
}

// GetPlatform is one agent the person may see (its Update page).
func (pc *PlatformController) GetPlatform(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	p, err := pc.svc.GetPlatform(c.Request.Context(), id, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (pc *PlatformController) SavePlatform(c *gin.Context) {
	access, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	var id *uuid.UUID
	if c.Param("id") != "" {
		parsed, ok := paramID(c)
		if !ok {
			return
		}
		id = &parsed
	}
	var in services.PlatformInput
	if !bindJSON(c, &in) {
		return
	}
	actor, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Putting the agent in groups or taking it out changes those groups: it needs Update Group
	// Agent Assignments too.
	if in.Groups != nil && !access.Can(permissions.PlatformAssignmentsEdit) {
		c.JSON(http.StatusForbidden, gin.H{"error": "changing an agent’s groups needs Update Group Agent Assignments"})
		return
	}
	p, groups, err := pc.svc.SavePlatformWithGroups(c.Request.Context(), id, in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	details := map[string]interface{}{
		"code": p.Code, "name": p.Name, "default_rate": p.DefaultRate.String(), "default_currency": p.DefaultCurrency, "is_active": p.IsActive,
	}
	if groups.Any() {
		details["groups"] = groups
	}
	if p.BranchID != nil {
		details["branch_id"] = p.BranchID.String()
	}
	if !p.IsActive && strings.TrimSpace(in.Reason) != "" {
		details["reason"] = strings.TrimSpace(in.Reason)
	}
	pc.record(c, actor, "platform.save", "platform", p.ID.String(), nil, details)
	status := http.StatusOK
	if id == nil {
		status = http.StatusCreated
	}
	// counted_weeks: weeks saved before it was put in a group that count in it now; reworked_weeks:
	// saved weeks worked out again at its new rates.
	c.JSON(status, struct {
		*models.Platform
		CountedWeeks  int `json:"counted_weeks,omitempty"`
		ReworkedWeeks int `json:"reworked_weeks,omitempty"`
	}{p, groups.Weeks, groups.Reworked})
}

// DeletePlatform removes an agent that was never used (a used one can only be turned off).
func (pc *PlatformController) DeletePlatform(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	p, err := pc.svc.DeletePlatformIn(c.Request.Context(), id, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actorOf(c), "platform.delete", "platform", id.String(), nil, map[string]interface{}{
		"code": p.Code, "name": p.Name, "default_currency": p.DefaultCurrency,
	})
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// ===== Partners =====

func (pc *PlatformController) ListPartners(c *gin.Context) {
	rows, err := pc.svc.ListPartners(c.Request.Context(), c.Query("include_inactive") == "true")
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

func (pc *PlatformController) SavePartner(c *gin.Context) {
	var id *uuid.UUID
	if c.Param("id") != "" {
		parsed, ok := paramID(c)
		if !ok {
			return
		}
		id = &parsed
	}
	var in services.PartnerInput
	if !bindJSON(c, &in) {
		return
	}
	actor, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	p, err := pc.svc.SavePartner(c.Request.Context(), id, in)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actor, "partner.save", "partner", p.ID.String(), nil, map[string]interface{}{
		"name": p.Name, "phone": p.Phone, "default_rate": p.DefaultRate.String(), "is_active": p.IsActive,
	})
	status := http.StatusOK
	if id == nil {
		status = http.StatusCreated
	}
	c.JSON(status, p)
}

// ===== Assignments =====

func (pc *PlatformController) ListAssignments(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	// No branch_id: the person's own group.
	branchID, ok := pc.ownGroupUnless(c, scope)
	if !ok {
		return
	}
	platformID, ok := queryID(c, "platform_id")
	if !ok {
		return
	}
	rows, err := pc.svc.ListAssignments(c.Request.Context(), services.AssignmentFilter{BranchID: branchID, PlatformID: platformID, Status: c.Query("status")}, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

// partnerDetails lists an assignment's partners for the audit log as "partner id: %".
func partnerDetails(list []models.AssignmentPartner) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.PartnerID.String() + ": " + p.CommissionRate.String()
	}
	return out
}

func assignmentDetails(a *services.Assignment) map[string]interface{} {
	return map[string]interface{}{
		"branch_id": a.BranchID.String(), "platform_id": a.PlatformID.String(), "partners": partnerDetails(a.Partners),
		"commission_rate": a.CommissionRate.String(), "currency": a.Currency, "starts_on": a.StartsOn.Format("2006-01-02"),
	}
}

func (pc *PlatformController) Assign(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	var in services.AssignmentInput
	if !bindJSON(c, &in) {
		return
	}
	actor, _ := getActorID(c)
	list, err := pc.svc.Assign(c.Request.Context(), in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	for i := range list {
		a := list[i]
		pc.record(c, actor, "platform.assignment.create", "hq_platform_assignment", a.ID.String(), &a.BranchID, assignmentDetails(&a))
	}
	c.JSON(http.StatusCreated, gin.H{"items": list})
}

func (pc *PlatformController) UpdateAssignment(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in services.AssignmentUpdateInput
	if !bindJSON(c, &in) {
		return
	}
	actor, _ := getActorID(c)
	before, after, err := pc.svc.UpdateAssignment(c.Request.Context(), id, in, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actor, "platform.assignment.update", "hq_platform_assignment", id.String(), &after.BranchID, map[string]interface{}{
		"before": map[string]interface{}{"partners": partnerDetails(before.Partners), "commission_rate": before.CommissionRate.String(), "currency": before.Currency},
		"after":  map[string]interface{}{"partners": partnerDetails(after.Partners), "commission_rate": after.CommissionRate.String(), "currency": after.Currency},
	})
	c.JSON(http.StatusOK, after)
}

func (pc *PlatformController) EndAssignment(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in services.AssignmentEndInput
	if !bindJSON(c, &in) {
		return
	}
	actor, _ := getActorID(c)
	a, err := pc.svc.EndAssignment(c.Request.Context(), id, in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	details := assignmentDetails(a)
	details["ends_on"], details["reason"] = a.EndsOn.Format("2006-01-02"), a.EndReason
	pc.record(c, actor, "platform.assignment.end", "hq_platform_assignment", id.String(), &a.BranchID, details)
	c.JSON(http.StatusOK, a)
}

func (pc *PlatformController) MoveAssignment(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in services.AssignmentMoveInput
	if !bindJSON(c, &in) {
		return
	}
	actor, _ := getActorID(c)
	ended, created, err := pc.svc.MoveAssignment(c.Request.Context(), id, in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	details := assignmentDetails(created)
	details["from_assignment"], details["from_branch_id"], details["from_ends_on"] = ended.ID.String(), ended.BranchID.String(), ended.EndsOn.Format("2006-01-02")
	pc.record(c, actor, "platform.assignment.move", "hq_platform_assignment", created.ID.String(), &created.BranchID, details)
	c.JSON(http.StatusOK, gin.H{"ended": ended, "created": created})
}

// ===== Groups (titled sets of agents) =====

func groupDetails(g *services.GroupRow) map[string]interface{} {
	agents := make([]string, len(g.Agents))
	for i, a := range g.Agents {
		agents[i] = a.PlatformCode
	}
	partners := make([]string, len(g.Partners))
	for i, p := range g.Partners {
		partners[i] = p.PartnerID.String() + ": " + p.CommissionRate.String()
	}
	details := map[string]interface{}{"title": g.Title, "agents": agents, "partners": partners, "currency": g.Currency, "is_active": g.IsActive}
	// Weeks saved before an agent was put in the group that count in it now; saved weeks worked out
	// again at the new rates.
	if g.CountedWeeks > 0 {
		details["counted_weeks"] = g.CountedWeeks
	}
	if g.ReworkedWeeks > 0 {
		details["reworked_weeks"] = g.ReworkedWeeks
	}
	return details
}

func (pc *PlatformController) ListGroups(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	// No branch_id: the person's own group (branch); branch_id=all: every branch in their scope.
	allBranches := c.Query("branch_id") == "all"
	var branchID *uuid.UUID
	if allBranches {
		branchID = &uuid.UUID{}
	} else if branchID, ok = pc.ownGroupUnless(c, scope); !ok {
		return
	}
	platformID, ok := queryID(c, "platform_id")
	if !ok {
		return
	}
	page, size, on := listPage(c)
	f := services.GroupFilter{PlatformID: platformID, Status: c.Query("status"), Search: c.Query("search"), AllBranches: allBranches}
	if on {
		f.Page, f.PageSize = page, size
	}
	rows, total, err := pc.svc.ListGroupsPage(c.Request.Context(), *branchID, f, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pagedJSON(c, orEmpty(rows), total, page, size, on)
}

func (pc *PlatformController) CreateGroup(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	var in services.GroupInput
	if !bindJSON(c, &in) {
		return
	}
	actor := actorOf(c)
	g, err := pc.svc.CreateGroup(c.Request.Context(), in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actor, "platform.group.create", "assignment_group", g.ID.String(), &g.BranchID, groupDetails(g))
	c.JSON(http.StatusCreated, g)
}

func (pc *PlatformController) UpdateGroup(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in services.GroupUpdateInput
	if !bindJSON(c, &in) {
		return
	}
	actor := actorOf(c)
	before, after, err := pc.svc.UpdateGroup(c.Request.Context(), id, in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actor, "platform.group.update", "assignment_group", id.String(), &after.BranchID, map[string]interface{}{
		"before": groupDetails(before), "after": groupDetails(after),
	})
	c.JSON(http.StatusOK, after)
}

type groupActiveInput struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

func (pc *PlatformController) SetGroupActive(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in groupActiveInput
	if !bindJSON(c, &in) {
		return
	}
	g, err := pc.svc.SetGroupActive(c.Request.Context(), id, *in.IsActive, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	action := "platform.group.turn_off"
	if g.IsActive {
		action = "platform.group.turn_on"
	}
	pc.record(c, actorOf(c), action, "assignment_group", id.String(), &g.BranchID, groupDetails(g))
	c.JSON(http.StatusOK, g)
}

func (pc *PlatformController) DeleteGroup(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	g, err := pc.svc.DeleteGroup(c.Request.Context(), id, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actorOf(c), "platform.group.delete", "assignment_group", id.String(), &g.BranchID, groupDetails(g))
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// ===== Weekly win/lose =====

func (pc *PlatformController) Formula(c *gin.Context) {
	current, err := pc.svc.Formula(c.Request.Context())
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"current": current, "versions": winlose.Versions()})
}

func (pc *PlatformController) WeekSheet(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	// An entry by entry_id; without one, a new entry for the week of week_start (no branch_id: in the
	// person's own group).
	actor, _ := getActorID(c)
	sheet, err := pc.svc.EntrySheet(c.Request.Context(), c.Query("entry_id"), c.Query("week_start"), c.Query("branch_id"), scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, sheet)
}

func (pc *PlatformController) Preview(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	var in services.WeekEntryInput
	if !bindJSON(c, &in) {
		return
	}
	actor, _ := getActorID(c)
	rows, err := pc.svc.Preview(c.Request.Context(), in, scope, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

func (pc *PlatformController) SaveWeek(c *gin.Context) {
	access, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	var in services.WeekEntryInput
	if !bindJSON(c, &in) {
		return
	}
	actor, _ := getActorID(c)
	sheet, corrections, err := pc.svc.SaveWeek(c.Request.Context(), in, scope, access.Can, actor)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	// Each changed saved week goes in the audit log with its old and new figures (or title).
	for _, cr := range corrections {
		action, entity := "winlose.correct", "weekly_platform_result"
		if cr.PlatformCode == "" {
			action, entity = "winlose.entry.rename", "weekly_entry"
		}
		pc.record(c, actor, action, entity, cr.ResultID.String(), &sheet.BranchID, map[string]interface{}{
			"week_start": sheet.WeekStart, "branch": sheet.BranchCode, "platform": cr.PlatformCode, "before": cr.Before, "after": cr.After, "reason": in.Reason,
		})
	}
	c.JSON(http.StatusOK, sheet)
}

type entryDeleteInput struct {
	Reason string `json:"reason" binding:"max=512"`
}

// DeleteEntry removes an entry and its week's amounts; the audit log keeps them and the reason.
func (pc *PlatformController) DeleteEntry(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in entryDeleteInput
	if !bindJSON(c, &in) {
		return
	}
	entry, removed, err := pc.svc.DeleteEntry(c.Request.Context(), id, in.Reason, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	amounts := make([]map[string]string, len(removed))
	for i, r := range removed {
		amounts[i] = map[string]string{"platform_id": r.PlatformID.String(), "currency": r.Currency, "win_lose": r.WinLose.String(), "commission": r.Commission.String(), "total": r.JamePS.String()}
	}
	pc.record(c, actorOf(c), "winlose.entry.delete", "weekly_entry", id.String(), &entry.BranchID, map[string]interface{}{
		"title": entry.Title, "week_start": entry.WeekStart.Format("2006-01-02"), "amounts": amounts, "reason": in.Reason,
	})
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

type entryReactionInput struct {
	Reaction string `json:"reaction" binding:"required,max=8"`
}

// ReactToEntry sets the person's ✅ (yes) or ❌ (no), or 📤 (sent) beside it, on an entry; the same
// again takes it back. The audit log has what was tapped and the person's reaction of that kind now.
func (pc *PlatformController) ReactToEntry(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in entryReactionInput
	if !bindJSON(c, &in) {
		return
	}
	entry, now, reactions, err := pc.svc.ReactToEntry(c.Request.Context(), id, in.Reaction, scope, actorOf(c))
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actorOf(c), "winlose.entry.react", "weekly_entry", id.String(), &entry.BranchID, map[string]interface{}{
		"title": entry.Title, "week_start": entry.WeekStart.Format("2006-01-02"), "reaction": now, "tapped": in.Reaction,
	})
	c.JSON(http.StatusOK, gin.H{"reaction": now, "reactions": reactions})
}

// ReactToGroup sets the person's ✅ (yes) or ❌ (no), or 📤 (sent), on a group in an entry (Group
// summary); the same again takes it back.
func (pc *PlatformController) ReactToGroup(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var in entryReactionInput
	if !bindJSON(c, &in) {
		return
	}
	entry, group, now, reactions, err := pc.svc.ReactToGroup(c.Request.Context(), id, groupID, in.Reaction, scope, actorOf(c))
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actorOf(c), "winlose.group.react", "weekly_entry", id.String(), &entry.BranchID, map[string]interface{}{
		"title": entry.Title, "week_start": entry.WeekStart.Format("2006-01-02"), "group_id": group.ID.String(), "group": group.Title, "reaction": now, "tapped": in.Reaction,
	})
	c.JSON(http.StatusOK, gin.H{"reaction": now, "reactions": reactions})
}

// ReactToAgent sets the person's ✅ (yes) or ❌ (no), or 📤 (sent), on an agent in an entry (Group
// summary: with or without a result there); the same again takes it back.
func (pc *PlatformController) ReactToAgent(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	platformID, err := uuid.Parse(c.Param("platform_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}
	var in entryReactionInput
	if !bindJSON(c, &in) {
		return
	}
	entry, agent, now, reactions, err := pc.svc.ReactToAgent(c.Request.Context(), id, platformID, in.Reaction, scope, actorOf(c))
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pc.record(c, actorOf(c), "winlose.agent.react", "weekly_entry", id.String(), &entry.BranchID, map[string]interface{}{
		"title": entry.Title, "week_start": entry.WeekStart.Format("2006-01-02"), "platform_id": agent.ID.String(), "agent": agent.Code, "reaction": now, "tapped": in.Reaction,
	})
	c.JSON(http.StatusOK, gin.H{"reaction": now, "reactions": reactions})
}

// ReactToResult sets the person's ✅ (yes) or ❌ (no), or 📤 (sent), on an agent's result in an entry
// (the agent in that entry); the same again takes it back.
func (pc *PlatformController) ReactToResult(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in entryReactionInput
	if !bindJSON(c, &in) {
		return
	}
	result, now, reactions, err := pc.svc.ReactToResult(c.Request.Context(), id, in.Reaction, scope, actorOf(c))
	if err != nil {
		writePlatformError(c, err)
		return
	}
	details := map[string]interface{}{"platform_id": result.PlatformID.String(), "week_start": result.WeekStart.Format("2006-01-02"), "reaction": now, "tapped": in.Reaction}
	if result.EntryID != nil {
		details["entry_id"] = result.EntryID.String()
	}
	pc.record(c, actorOf(c), "winlose.result.react", "weekly_platform_result", id.String(), &result.BranchID, details)
	c.JSON(http.StatusOK, gin.H{"reaction": now, "reactions": reactions})
}

func (pc *PlatformController) GetGroup(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	g, err := pc.svc.GroupByID(c.Request.Context(), id, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	c.JSON(http.StatusOK, g)
}

// ListEntries is Data Entry's list: the saved weeks of the person's groups, newest first.
func (pc *PlatformController) ListEntries(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	page, size, on := listPage(c)
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	f := services.EntryFilter{BranchID: branchID, Search: c.Query("search"), CreatedFrom: c.Query("created_from"), CreatedTo: c.Query("created_to")}
	if on {
		f.Page, f.PageSize = page, size
	}
	rows, total, err := pc.svc.ListEntriesPage(c.Request.Context(), f, scope)
	if err != nil {
		writePlatformError(c, err)
		return
	}
	pagedJSON(c, orEmpty(rows), total, page, size, on)
}

// ===== Reports =====

func (pc *PlatformController) Report(c *gin.Context) {
	access, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	partnerID, ok := queryID(c, "partner_id")
	if !ok {
		return
	}
	groupID, ok := queryID(c, "group_id")
	if !ok {
		return
	}
	platformID, ok := queryID(c, "platform_id")
	if !ok {
		return
	}
	entryID, ok := queryID(c, "entry_id")
	if !ok {
		return
	}
	report, err := pc.svc.Report(c.Request.Context(), services.ReportFilter{DateFrom: c.Query("date_from"), DateTo: c.Query("date_to"), BranchID: branchID,
		PartnerID: partnerID, GroupID: groupID, PlatformID: platformID, ByWeek: c.Query("by_week") == "true", ByEntry: c.Query("by_entry") == "true", EntryID: entryID,
		Everyone: c.Query("everyone") == "true"},
		scope, access.Can(permissions.WinLoseReportsConsolidated), actorOf(c))
	if err != nil {
		writePlatformError(c, err)
		return
	}
	if !access.Can(permissions.WinLoseReportsRates) {
		report.HideRates()
	}
	c.JSON(http.StatusOK, report)
}

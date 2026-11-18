#!/bin/sh
# Updates the app on this server to a commit of main, rebuilds it and waits until
# the API answers. GitHub Actions runs it after CI passes on finance-pro
# (.github/workflows/deploy.yml), through an SSH key that can run nothing else.
# Setup: docs/DEPLOY_HOSTINGER.md, "Deploying automatically".
#
# Install or update it on the server (as root, from the clone):
#   install -m 755 deploy/finance-deploy.sh /usr/local/bin/finance-deploy
# It runs from that copy, never from the clone, because the clone changes
# while it runs.
#
# By hand: finance-deploy deploys the newest main.
set -eu

# The clone on this server, with docker-compose.yml and .env next to it.
APP_DIR=/var/www/finance_pro

# One deploy at a time: a second one waits here for the first to finish.
exec 9>/var/lock/finance-deploy.lock
flock 9

# Over SSH, the only thing the caller sends is the commit to deploy (the
# authorized_keys entry puts it in SSH_ORIGINAL_COMMAND). Anything other than
# a full commit id is refused, so the key can't be used to run commands.
commit=${SSH_ORIGINAL_COMMAND:-}
if [ -n "$commit" ]; then
	case $commit in
	*[!0-9a-f]*)
		echo "finance-deploy: expected a 40-character commit id" >&2
		exit 2
		;;
	esac
	if [ ${#commit} -ne 40 ]; then
		echo "finance-deploy: expected a 40-character commit id" >&2
		exit 2
	fi
fi

cd "$APP_DIR"
git fetch --quiet origin main
# Only ever moves forward along main: a commit that isn't on main, or a clone
# with edits or commits of its own, stops the deploy here. A commit older than
# the one already deployed changes nothing.
git merge --quiet --ff-only "${commit:-origin/main}"
echo "finance-deploy: code is at $(git log --oneline -1)"

# .env and data/ are not in git, so the settings and the data stay as they are.
# If the build fails, the running containers are left as they were.
docker compose up -d --build

# The API answers /ready once it has started, updated the tables and reached
# the database (the API listens on 8081 inside the app container).
tries=0
until docker compose exec -T app wget -q -O /dev/null http://127.0.0.1:8081/ready 2>/dev/null; do
	tries=$((tries + 1))
	if [ "$tries" -ge 90 ]; then
		echo "finance-deploy: the API didn't answer within 3 minutes. See: cd $APP_DIR && docker compose logs --tail=50 app" >&2
		exit 1
	fi
	sleep 2
done

# Each build leaves the previous image behind (several hundred MB a deploy).
# Remove this app's unused ones: only images labelled finance.image (set in
# docker-compose.yml), so nothing else on the server is touched.
docker image prune -f --filter label=finance.image >/dev/null
echo "finance-deploy: done, the API is answering"

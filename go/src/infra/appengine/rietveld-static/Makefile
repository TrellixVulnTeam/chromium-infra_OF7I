# Deploy gentasks to prod
.PHONY: gentasks
gentasks:
	gae.py upload --app-id chromiumcodereview-hr --app-dir . gentasks

.PHONY: deploy
deploy:
	gae.py upload --app-id chromiumcodereview-hr --app-dir . default
	gae.py upload --app-id chromiumcodereview-private --app-dir . default


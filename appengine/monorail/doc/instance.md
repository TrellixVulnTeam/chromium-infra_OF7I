# Creating a new Monorail instance

1.  Create new GAE apps for production and staging.
1.  Configure GCP billing.
1.  Fork settings.py and configure every part of it, especially trusted domains
    and "all" email settings.
    1.  Change num_logical_shards to the number of read replicas you want.
    1.  You might want to also update `*/*_constants.py` files.
1.  Create new primary DB and an appropriate number of read replicas for prod
    and staging.
    1.  Set up IP address and configure admin password and allowed IP addr.
        [Instructions](https://cloud.google.com/sql/docs/mysql-client#configure-instance-mysql).
    1.  Set up backups on primary DB. The first backup must be created before
        you can configure replicas.
1.  Set up log saving to bigquery or something.
1.  Set up monitoring and alerts.
1.  Set up attachment storage in GCS.
1.  Set up spam data and train models.
1.  Fork and customize some of HTML in templates/framework/master-header.ezt,
    master-footer.ezt, and some CSS to give the instance a visually different
    appearance.
1.  Get From-address allowlisted so that the "View issue" link in Gmail/Inbox
    works.
1.  Set up a custom domain with SSL and get that configured into GAE. Make sure
    to have some kind of reminder system set up so that you know before cert
    expire.
1.  Configure the API. Details? Allowed clients are now configured through
    luci-config, so that is a whole other thing to set up. (Or, maybe decide not
    to offer any API access.)
1.  Gain permission to sync GGG user groups. Set up borgcron job to sync user
    groups. Configure that job to hit the API for your instance. (Or, maybe
    decide not to sync any user groups.)
1.  Monorail does not not access any internal APIs, so no allowlisting is
    required.
1.  For projects on code.google.com, coordinate with that team to set flags to
    do per-issue redirects from old project to new site. As each project is
    imported, set it's moved-to field.

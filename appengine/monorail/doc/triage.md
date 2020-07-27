# Monorail Triage Guide (go/monorail-triage)

Monorail is a tool that is actively used and maintained by
the Chromium community.  It is important that we look at
issues reported by our users and take appropriate actions.

For the full list of trooper responsibilities, see
[go/chops-workflow-trooper](http://go/chops-workflow-trooper).

## Triage Process

Look at each issue in the
[Monorail untriaged
queue](https://bugs.chromium.org/p/monorail/issues/list?q=&can=41013401) and
[Sheriffbot untriaged queue](http://crbug.com/?q=component%3DTools%3EStability%3ESheriffbot%20status%3Auntriaged&can=2)
and do the following:

* If the issue is unintelligible or empty, flag the issue as spam.
  * Check the user stats at bugs.chromium.org/u/{user\_email}/updates, and if none of their
  activities on the site are serious or valid, ban them as spammer.
* Move issues to the correct project (eg. "chromium") if misfiled.
* Apply the correct `type:` label.
* If the bug is caused by someone else's changes or if the bug is part of the feature
  clearly owned by one person, assign the issue to that person and set
  `status:Assigned`.
* Validate the issue and make it clear and actionable
  * If not actionable or reproducible, mark issue as `status:WontFix`
  * If the issue requires more information from reporter, add the label
    `needs:Feedback` and ask the reporter for more information.
* Update issue `Pri-` label
  * If the issue is a Monorail API Request, set `Pri-1`
  * If the issue is a security or privacy issue, set `Pri-1`
  * Refer to [Priorities](#Priorities) for all other cases.
* If the issue is `Pri-0` or `Pri-1`
  * If `Pri-0`: assign self as `owner`, mark `status:Started`, notify team leads, follow
    the IRM process as Incident Commander.
  * If `Pri-1`: assign self as owner, mark `status:Started` and work to resolve the
    issue. Find another owner and make a formal handoff if you are not able to
    address.
* If an issue has been `needs:Feedback` for more than 7 days without response, mark
  as `status:WontFix` with an explanatory comment.
* Otherwise, mark issue as `status:Available`

## Priorities

* `Pri-0`: Critical issue causing failures in production. Major functionality broken
  that renders a feature unusable for all customers.
* `Pri-1`: Urgent; the issue is blocking a user from getting their job done. Defect
  causing functional regression in production. Production issue impacting other
  customers. Any type of security or privacy problem. Finally, any workflow
  administrative tasks that have been officially asked to the trooper to handle,
  that includes very explicitly: Monorail API access request, Sheriffbot testing &
  deployment, and Hotlist removal.
* `Pri-2`: Important; tied to OKRs or near term upcoming release. Bug that should be
  addressed in one of the next few releases.
* `Pri-3`: We feel your pain: the team would like to fix this, but lacks the resources
  to do this soon. Desirable feature or enhancement not on the near-term roadmap.
  Defects that are not regressions, have workarounds, and affect few users.
* `Pri-4`: Ponies and icebox. Unfortunate: it's a legitimate issue, but the team never
  plans to fix this.

# Monorail Source Code Organization

[TOC]

## Overview

Monorail's source code organization evolved from the way that source
code was organized for Google Code (code.google.com).  That site
featured a suite of tools with code for each tool in a separate
directory.  Roughly speaking, the majority of the code was organized
to match the information architecture of the web site.  Monorail keeps
that general approach, but makes a distinction between core issue
tracking functionality (in the `tracker` directory) and additional
features (in the `features` directory).

Also dating back to Google Code's 2005 design, the concept of a
"servlet" is key to Monorail's UI-centric source code organization.  A
servlet is a python class with methods to handle all functionality for
a given UI web page. Servlets handle the initial page rendering, any
form processing, and have related classes for any XHR calls needed for
interactive elements on a page.  Servlet's mix application business
logic, e.g., permission checks, with purely UI logic, e.g., screen
flow and echoing UI state in query string parameters.

From 2018 to 2020, the old servlet-oriented source code organization
is slowly being hollowed out and replaced with a more API-centric
implementation.  Server-side python code is gradually being shifted
from the `tracker`, `project`, `sitewide`, and `features` directories
to the `api` and `businesslogic` directories.  While more UI logic is
being shifted from python code into javascript code under
`static_src`.

Although Monorail's GAE app has several GAE services, we do not
organize the source code around GAE services because they each share a
significant amount of code.

## Source code dependency layers

At a high level, the code is organized into a few logical layers with
dependencies going downward:

App-integration layer

*  The main program `monorailapp.py` that ties all the servlets together.

*  All GAE configuration files, e.g., `app.yaml` and `cron.yaml`.

Request handler layer

*  This including servlets, inbound email, Endpoints, and rRPC.

*  These files handle a request from a user, including syntactic
   validation of request fields, and formatting of the response.

Business logic layer

*  This layer does the main work of handling a request, including
   semantic validation of whether the request is valid, permission
   checking, coordinating calls to the services layer, and kicking off
   any follow-up tasks such as sending notification.

*  Much of the content of `*_helper.py` and `*_bizobj.py` files also
   belong to this layer, even though it has not been moved here as of
   2019.

Services layer

*  This layer include our object-relational-mapping logic.

*  It also manages connections to backends that we use other than the
   database, for example full-text search.

Framework layer

*  This has code that provides widely used functionality and systemic
   features, e.g.,`sql.py` and `xsrf.py`.

Asset layer

*  These are low-level files that can be included from anywhere in the
   code, and should not depend on anything else.  For example,
   `settings.py`, various `*_constants.py` files, and protobuf
   definitions.


## Source code file and directories by layer

App-integration layer

*  `monorailapp.py`: The main program that loads all web app request
   handlers.

*  `registerpages.py`: Code to register specific request handlers at
   specific URLs.

*  `*.yaml`: GAE configuration files

Request handler layer

*  `tracker/*.py`: Servlets for core issue tracking functionality.

*  `features/*.py`: Servlets for issue tracking features that are not
   core to the product.

*  `project/*.py`: Servlets for creating and configuring projects and
   memberships.

*  `sitewide/*.py`: Servlets for user profile pages, the site home
   page, and user groups.

*  `templates/*/*.ezt`: Template files for old web UI page generation.

*  `api/*.py`: pRPC API request handlers.

*  `services/api_svc_v1.py`: Endpoints request handlers.

*  `features/inboundemail.py`: Inbound email request handlers and bounces.

*  `features/notify.py`: Email notification task handlers.


Business logic layer

*  `businesslogic/work_env.py`:  Internal API for monorail.

*  `*/*_bizobj.py` and `*/*_helpers.py*` files: Business logic that was
   written for servlets but that is gradually being used only through
   work_env.

Services layer

*  `schema/*.sql`:  SQL database table definitions.

*  `services/service_manager.py`: Simple object to hold all service objects.

*  `services/caches.py` and `cachemanager.py`: RAM and memcache caches
   and distributed invalidation.

*  `services/issues_svc.py`: DB persistence for issues, comments, and
   attachments

*  `services/user_svc.py`: Persistence for user accounts.

*  `services/usergroup_svc.py`: Persistence for user groups.

*  `services/features_svc.py`: Persistence for hotlists, saved queries,
   and filter rules.

*  `services/chart_svc.py`: Persistence for issue snapshots and
   charting queries.

*  `services/secrets_svc.py`: Datastore code for key used to generate
   XSRF and recaptcha tokens.

*  `services/project_svc.py`: Persistence for projects and members.

*  `services/config_svc.py`: Persistence for issue tracking
   configuration in a project, except templates.

*  `services/client_config_svc.py`: Persistence for API whitelist.

*  `services/tracker_fulltext.py`: Connection to GAE fulltext search
   index.

*  `services/template_svc.py`: Persistence for issue templates.

*  `services/star_svc.py`: Persistence for all types of stars.

*  `services/spam_svc.py`: Persistence for abuse flags and spam verdicts.

*  `services/ml_helpers.py`: Utilities for working with ML Engine backend.

*  `search/*`: frontend and backend code for sharded issue search and
   result set caching.


Framework layer

*  `framework/sql.py`: SQL DB table managers and safe SQL statement
   generation.

*  `framework/servlet.py` and `jsonfeed.py`:  Base classes for servlets.

*  `framework/warmup.py`: Trivial servlet needed for GAE warmup feature.

*  `framework/permissions.py`: Permissionset objects and permission
   checking functions.

*  `framework/authdata.py`, `monorailrequest.py`, and
   `monorailcontext.py`: objects that represent information about the
   incoming request.

*  `framework/xsrf.py`, `banned.py`, and `captcha.py`: Anti-abuse utilities.

*  `testing/*.py`: Utilities for python unit tests.

Asset layer

*  `settings.py`: Server instance configuration.

*  `framework/urls.py`: Constants for web UI URLs.

*  `framework/exceptions.py`: python exceptions used throughout the code.

*  `framework/framework_constants.py`: Implementation-oriented constants.

*  `proto/*.proto`: ProtoRPC definitions for internal representation of
   business objects.

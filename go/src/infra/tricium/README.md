# Tricium

Tricium is a code analysis platform for Chromium and related projects;
Tricium enables automatic analysis with comments posted to code review.

In 2022, there are plans to merge the functionality of Tricium into the new
[Change Verifier
service](https://source.chromium.org/chromium/infra/infra/+/main:go/src/go.chromium.org/luci/cv/README.md).
This planned merge will not affect existing analyzers, although this codebase
will be deleted; no changes will be required from analyzer owners or
developers.

But until the merge, Tricium is in "maintenance mode" and we don't plan to make
significant changes at this time.

**Contact**:

*   Tricium team discussion: `tricium-dev@google.com`.
*   Public mailing list which can be used for any Chrome Ops related discussion:
    `infra-dev@chromium.org`, or LUCI development discussion: `luci-eng@google.com`.

**Documentation**:

*   [User guide](docs/user-guide.md): how to configure Tricium for your project.
*   [Analyzer development guide](docs/contribute.md): how to add a new Tricium analyzer.

**Locations**:

*   Service code:
    [here](https://chromium.googlesource.com/infra/infra/+/master/go/src/infra/tricium/),
    [code search](https://cs.chromium.org/chromium/infra/go/src/infra/tricium/)
*   PolyGerrit Plugin:
    [infra/gerrit-plugins/tricium](https://chromium.googlesource.com/infra/gerrit-plugins/tricium)
*   Recipe module: [tricium recipe module in recipes-py](https://source.chromium.org/chromium/infra/infra/+/main:recipes-py/recipe_modules/tricium/api.py)
*   Individual project recipes live with each project, e.g. [chromium "simple" analyzers recipe](https://source.chromium.org/chromium/chromium/tools/build/+/main:recipes/recipes/tricium_simple.py).
*   Production Server URL: [tricium-prod.appspot.com](https://tricium-prod.appspot.com/)
*   Dev Server URL: [tricium-dev.appspot.com](https://tricium-dev.appspot.com/)

# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api


class DockerApi(recipe_api.RecipeApi):
  """Provides steps to connect and run Docker images."""

  def __init__(self, *args, **kwargs):
    super(DockerApi, self).__init__(*args, **kwargs)
    self._config_file = None
    self._project = None
    self._server = None

  def ensure_installed(self, **kwargs):
    """Checks that the docker binary is in the PATH.

    Raises StepFailure if binary is not found.
    """
    try:
      self.m.step('ensure docker installed', ['which', 'docker'], **kwargs)
    except self.m.step.StepFailure as f:
      f.result.presentation.step_text = (
          'Error: is docker not installed or not in the PATH')
      raise

  def get_version(self):
    """Returns Docker version installed or None if failed to detect."""
    docker_version_step = self(
        'version',
        stdout=self.m.raw_io.output(),
        step_test_data=(
            lambda: self.m.raw_io.test_api.stream_output('Version: 1.2.3')))
    for line in docker_version_step.stdout.splitlines():
      line = line.strip().lower()
      if line.startswith('version: '):
        version = line[len('version: '):]
        docker_version_step.presentation.step_text = version
        return version
    else:
      docker_version_step.presentation.step_text = 'Version unknown?'
      return None

  def login(self,
            server='gcr.io',
            project='chromium-container-registry',
            service_account=None,
            step_name=None,
            **kwargs):
    """Connect to a Docker registry.

    This step must be executed before any other step in this module that
    requires authentication.

    Args:
      server: GCP container registry to pull images from. Defaults to 'gcr.io'.
      project: Name of the Cloud project where Docker images are hosted.
      service_account: service_account.api.ServiceAccount used for
          authenticating with the container registry. Defaults to the task's
          associated service account.
      step_name: Override step name. Default is 'docker login'.
    """
    # We store config file in the cleanup dir to ensure that it is deleted after
    # the build finishes running. This way no subsequent builds running on the
    # same bot can re-use credentials obtained below.
    self._config_file = self.m.path['cleanup'].join('.docker')
    self._project = project
    self._server = server
    if not service_account:
      service_account = self.m.service_account.default()
    token = service_account.get_access_token(
        ['https://www.googleapis.com/auth/cloud-platform'])
    self.m.python(
        step_name or 'docker login',
        self.resource('docker_login.py'),
        args=[
            '--server',
            server,
            '--service-account-token-file',
            self.m.raw_io.input(token),
            '--config-file',
            self._config_file,
        ],
        **kwargs)

  def pull(self, image, step_name=None):
    """Pull a docker image from a remote repository.

    Args:
      image: Name of the image to pull.
      step_name: Override step name. Default is 'docker pull'.
    """
    assert self._config_file, 'Did you forget to call docker.login?'
    self.m.step(
        step_name or 'docker pull %s' % image,
        [
            'docker', '--config', self._config_file, 'pull',
            '%s/%s/%s' % (self._server, self._project, image)
        ],
    )

  def run(self,
          image,
          step_name=None,
          cmd_args=None,
          dir_mapping=None,
          env=None,
          inherit_luci_context=False,
          **kwargs):
    """Run a command in a Docker image as the current user:group.

    Args:
      image: Name of the image to run.
      step_name: Override step name. Default is 'docker run'.
      cmd_args: Used to specify command to run in an image as a list of
          arguments. If not specified, then the default command embedded into
          the image is executed.
      dir_mapping: List of tuples (host_dir, docker_dir) mapping host
          directories to directories in a Docker container. Directories are
          mapped as read-write.
      env: dict of env variables.
      inherit_luci_context: Inherit current LUCI Context (including auth).
          CAUTION: removes network isolation between the container and the
          docker host. Read more https://docs.docker.com/network/host/.
    """
    assert self._config_file, 'Did you forget to call docker.login?'
    args = [
        '--config-file',
        self._config_file,
        '--image',
        '%s/%s/%s' % (self._server, self._project, image),
    ]

    if dir_mapping:
      for host_dir, docker_dir in dir_mapping:
        args.extend(['--dir-map', host_dir, docker_dir])

    for k, v in sorted((env or {}).iteritems()):
      args.extend(['--env', '%s=%s' % (k, v)])

    if inherit_luci_context:
      args.append('--inherit-luci-context')

    if cmd_args:
      args.append('--')
      args += cmd_args

    self.m.python(
        step_name or 'docker run',
        self.resource('docker_run.py'),
        args=args,
        **kwargs)

  def __call__(self, *args, **kwargs):
    """Executes specified docker command.

    Please make sure to use api.docker.login method before if specified command
    requires authentication.

    Args:
      args: arguments passed to the 'docker' command including subcommand name,
          e.g. api.docker('push', 'my_image:latest').
      kwargs: arguments passed down to api.step module.
    """
    cmd = ['docker']
    if '--config' not in args and self._config_file:
      cmd += ['--config', self._config_file]
    step_name = kwargs.pop('step_name', 'docker %s' % args[0])
    return self.m.step(step_name, cmd + list(args), **kwargs)

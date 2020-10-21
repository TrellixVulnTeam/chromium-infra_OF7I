FROM gcr.io/google-appengine/python

ADD . .

# Install chromeperf package with --editable, which avoids
# 'ZIP does not support timestamps before 1980' in cloudbuild
RUN pip3 install --upgrade setuptools gunicorn && \
    pip3 install --editable .

# Run as `gunicorn 'chromeperf.pinpoint.service:CreateApp()'`
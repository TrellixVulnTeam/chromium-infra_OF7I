FROM gcr.io/google-appengine/python

ADD . .

RUN pip3 install --upgrade setuptools gunicorn && \
    python3 setup.py install

# Run as `gunicorn 'chromeperf.pinpoint.service:CreateApp()'`
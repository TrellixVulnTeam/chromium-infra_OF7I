FROM python:3.8
WORKDIR /src
COPY requirements.txt ./
RUN pip3 install --no-cache-dir -r requirements.txt

COPY ./lstm.py ./

CMD [ "python", "./lstm.py" ]

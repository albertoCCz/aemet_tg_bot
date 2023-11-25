FROM python:3.12-slim-bookworm

VOLUME /usr/src/app/templates
VOLUME /usr/src/app/pdfs-registry

WORKDIR /usr/src/app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD [ "python", "./main.py" ]
# syntax=docker/dockerfile:1

FROM python:REPLACE_PYTHONVERSION-slim-buster

COPY . /app
COPY pyproject.toml /app

WORKDIR /app

RUN pip3 install poetry==REPLACE_POETRYVERSION
RUN poetry config virtualenvs.create false
RUN poetry install --no-dev

EXPOSE 50051
CMD ["python3", "-u", "/app/REPLACE_APPNAME/main.py"]
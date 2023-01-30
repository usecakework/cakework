# syntax=docker/dockerfile:1

FROM python:3.9-slim-buster

COPY . /app
COPY pyproject.toml /app

WORKDIR /app

RUN pip3 install poetry==1.3.2
RUN poetry config virtualenvs.create false
RUN poetry install --no-dev

EXPOSE 50051
CMD ["python3", "-u", "/app/react_example_backend/main.py"]
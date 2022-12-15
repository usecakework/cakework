# syntax=docker/dockerfile:1

FROM python:3.8-slim-buster

# COPY ./requirements_cakework.txt /app/requirements_cakework.txt
WORKDIR /app
COPY . /app
RUN ls -ltr
RUN echo "grpcio\nprotobuf\ncakework" >> /app/requirements.txt
RUN pip3 install -r /app/requirements.txt
ENV MODULE=submodule.activity
ENV ACTIVITY=activity
RUN python3 /app/register_activity.py
EXPOSE 50051
CMD ["python3", "-m", "cakework.activity_server"]
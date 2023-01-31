import subprocess
import os
import shutil
import sys
import inspect
import json

from concurrent import futures
from cakework import cakework_pb2
from cakework import cakework_pb2_grpc
from .task_server import TaskServer
import importlib
import logging

logging.basicConfig(level=logging.INFO)
class Cakework:
	def __init__(self, name, local=False): # TODO remove local when release as open source
		self.name = name
		self.local = local

		logging.info("Created project with name: " + self.name)

	def add_task(self, task):
		activity_server = TaskServer(task, self.local)
		activity_server.start()

def serve():
    port = '50051'
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=1)) # what should the default be?
    cakework_pb2_grpc.add_CakeworkServicer_to_server(cakework_pb2_grpc.Cakework(), server)
    server.add_insecure_port('[::]:' + port)
    server.start()
    logging.info("Server started, listening on " + port)
    server.wait_for_termination()
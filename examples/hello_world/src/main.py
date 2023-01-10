from cakework import Cakework
import time

def say_hello(name):
    time.sleep(5)
    return "hello " + name

if __name__ == "__main__":
    cakework = Cakework("hello_world")
    cakework.add_task(say_hello)

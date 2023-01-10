from cakework import Cakework
import time

def say_hello(name):
    time.sleep(5)
    return "hello " + name

if __name__ == "__main__":
    app = Cakework("hello_world")
    app.add_task(say_hello)

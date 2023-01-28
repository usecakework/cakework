from cakework import Cakework
import time

def say_hello(name):
    time.sleep(5)
    return "Hello " + name + "!"

if __name__ == "__main__":
    cakework = Cakework("REPLACE_APPNAME")
    cakework.add_task(say_hello)
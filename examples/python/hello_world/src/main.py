from cakework import App

def say_hello(name):
    return "hello " + name

if __name__ == "__main__":
    app = App("hello_world")
    app.add_task(say_hello)
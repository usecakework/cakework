from cakework import App

def say_hello(name):
    return "hello " + name

if __name__ == "__main__":
    app = App("myapp2")
    app.register_task(say_hello)
from cakework import Client

if __name__ == "__main__":
    client = Client("myapp2")
    result = client.say_hello(name="jessie")
    print("got result: " + result)

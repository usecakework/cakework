# todo PYNECONE ME

from cakework import Client

if __name__ == "__main__":
    client = Client("image_generation")
    result = client.generate_image(object="cute piece of cake", style="cartoon")
    print(result)

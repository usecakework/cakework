from cakework import Cakework
import banana_dev as banana
import base64
from io import BytesIO
from PIL import Image
import boto3
from nanoid import generate

AWS_ACCESS_KEY_ID = "AKIAYZ53GM6UJWTWCNPS"
AWS_SECRET_ACCESS_KEY = "S46BaUR1Iu6UWQrRz79VH8p2hqO1e84OM7xEzuWW"
AWS_BUCKET = "cakework-public-examples"
BANANA_API_KEY = "db4cff41-3568-4756-9c30-56d26c18647f"
BANANA_MODEL_KEY = "6daae4bd-0805-4ef8-9d51-be340d3f48bf"

# Run image generation model, returning the image as bytes.
# We are running stable diffusion hosted on banana.
def run_image_generation_model(prompt):
    model_inputs = {
        "prompt": prompt,
        "num_inference_steps": 50,
        "guidance_scale": 9,
        "height": 512,
        "width": 512,
        "seed": 3242
    }
    
    output = banana.run(BANANA_API_KEY, BANANA_MODEL_KEY, model_inputs)
    image_byte_string = output["modelOutputs"][0]["image_base64"]
    image_encoded = image_byte_string.encode('utf-8')
    image_bytes = BytesIO(base64.b64decode(image_encoded))
    return Image.open(image_bytes)

# Converts the image to a .jpg and uploads it to S3
def upload_image(image):
    s3 = boto3.client(
    's3',
    aws_access_key_id=AWS_ACCESS_KEY_ID,
    aws_secret_access_key=AWS_SECRET_ACCESS_KEY)

    id = generate(size=10)
    filename = id + ".png"

    file = BytesIO()
    image.save(file, "png")
    file.seek(0)

    s3.upload_fileobj(file, AWS_BUCKET, filename)
    return filename

# Generates a stable diffusion prompt from input
def generate_prompt(object, style):
    return object + " in " + style + " style "

# Run the image generation process and save to S3
def generate_image(object, style):
    prompt = generate_prompt(object, style)
    image = run_image_generation_model(prompt)
    s3_location = upload_image(image)

    return {
        "s3Location": s3_location
    }

if __name__ == "__main__":
    app = Cakework("image_generation")
    app.add_task(generate_image)

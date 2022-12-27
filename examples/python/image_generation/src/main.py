from cakework import App
import banana_dev as banana
import base64
from io import BytesIO
from PIL import Image
import boto3
from botocore.exceptions import ClientError
from nanoid import generate

AWS_ACCESS_KEY_ID = "YOUR_ACCESS_KEY_ID"
AWS_SECRET_ACCESS_KEY = "YOUR_SECRET_KEY"
AWS_BUCKET = "YOUR_AWS_BUCKET"
BANANA_API_KEY = "YOUR_API_KEY"
BANANA_MODEL_KEY = "YOUR_MODEL_KEY"

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
    try:
        prompt = generate_prompt(object, style)
        image = run_image_generation_model(prompt)
        s3_location = upload_image(image)
    except ClientError as clientError:
        return {
            "status": "FAILED",
            "error": clientError.response
        }

    return {
        "status":"SUCCEEDED",
        "s3Location": s3_location
    }

if __name__ == "__main__":
    app = App("image_generation")
    app.register_task(generate_image)

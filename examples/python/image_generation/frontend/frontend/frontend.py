from pcconfig import config

import pynecone as pc
from cakework import Client
import asyncio
import json

class State(pc.State):
    """The app state."""
    subject: str
    images = []

    def set_subject(self, subject):
        self.subject = subject

    def start_generate_image(self):
        # Start a Cakework task to generate an image
        client = Client("image_generation")
        request_id = client.generate_image(self.subject, "cartoon")

        with pc.session() as session:
            session.add(
                Image(
                    username=request_id,       
                )
            )
            session.commit()
        return self.poll_for_statuses

    async def poll_for_statuses(self):
        client = Client("image_generation")
        requests = []
        with pc.session() as session:
            requests = (
                session.query(Image)
                .all()
            )
        
        imageDict = {i[0]:i[len(i)-1] for i in self.images}
        for request in requests:
            image = []
            image.append(request.username)
            # Get status from Cakework
            status = client.get_status(request.username)
            image.append(status)
            if status == "SUCCEEDED":
                # Get the result from Cakework
                result = json.loads(client.get_result(request.username))
                download_url = "https://cakework-public-examples.s3.us-west-2.amazonaws.com/" + result["s3Location"]
                image.append(download_url)

            imageDict[request.username] = image

        self.images = list(imageDict.values())

        await asyncio.sleep(0.5)

        return self.poll_for_statuses

    def get_status(self, request_id):
        return

def index():
    return pc.center(
        pc.vstack(
            pc.heading("Make a Picture!", font_size="2em"),
            pc.spacer(),
            pc.text("What do you want to make?"),
            pc.editable(
                pc.editable_preview(),
                pc.editable_input(),
                placeholder="A cute piece of cake",
                on_submit=State.set_subject,
                width="100%",
                border="1px",
            ),
            pc.button("Go!", on_click=State.start_generate_image),
            pc.spacer(),
            pc.button("Poll For Statuses", on_click=State.poll_for_statuses),
            pc.data_table(
                data=State.images,
                columns=["ID", "Status", "URL"],
                pagination=True,
            )
        ),
        padding_top="10%",
    )

# TODO make it work with a request ID
class Image(pc.Model, table=True):
    username: str

# Add state and page to the app.
app = pc.App(state=State)
app.add_page(index)
app.compile()
"""Welcome to Pynecone! This file outlines the steps to create a basic app."""
from pcconfig import config

import pynecone as pc
from cakework import Client

# add status here somehow and reflect it in the UI
# show spinner
# show query status if a generation is already in progress (stuff survives through browser)
# etc
class State(pc.State):
    """The app state."""
    subject: str
    processing_status: str = "IN_PROGRESS"

    def set_subject(self, subject):
        self.subject = subject

    def generate_image(self):
        client = Client("image_generation")
        result = client.generate_image(object="cute piece of cake", style="cartoon")
        print(result)


def index():
    return pc.center(
        pc.vstack(
            pc.heading("Generate a Picture!", font_size="2em"),
            pc.editable(
                pc.editable_preview(),
                pc.editable_input(),
                placeholder="A piece of cake",
                on_submit=State.set_subject,
                width="100%",
                border="1px",
                margin="5px"
            ),
            pc.button("Get it done!", on_click=State.generate_image),
            pc.text(State.processing_status),
            spacing="1.5em",
            font_size="2em",
        ),
        padding_top="10%",
    )


# Add state and page to the app.
app = pc.App(state=State)
app.add_page(index)
app.compile()

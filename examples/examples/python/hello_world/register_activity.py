from cakework.app import App
from submodule.activity import activity

app = App("app")
app.register_activity(activity, "activity")
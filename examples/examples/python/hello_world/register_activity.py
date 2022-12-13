from sahale.app import App
from submodule.activity import activity

app = App("id", "app")
app.register_activity(activity, "activity")

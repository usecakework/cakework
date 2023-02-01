import { CakeworkApiClient } from "@cakework/fern_client";

class CakeworkClient {
    constructor(project, token) {
        this.project = project;
        this.token = token;
        this.client = new CakeworkApiClient({
            project: project,
        })
    }

    async run(task, params, compute) {     
        const run = {
            token: this.token,     
            body: {}
        };
        if (params != undefined) {
            run.body["parameters"] = params;
        }
        if (compute != undefined) {
            run.body["compute"] = params;
        }

        const runId = await this.client.client.run(this.project, task, run);
        return runId;
    }

    async getRunStatus(runId) {        
        const request = {
            token: this.token,     
        };
        const runStatus = await this.client.client.getRunStatus(runId, request);
        return runStatus;
    }

    async getRunResult (runId) {
        const request = {
            token: this.token,     
        };
        const runResult = await this.client.client.getRunResult(runId, request);
        return runResult;
    }
}

export default CakeworkClient;
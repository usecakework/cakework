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
        const request = {
            token: this.token,     
            body: {}
        };
        if (params != undefined) {
            request.body["parameters"] = params;
        }
        if (compute != undefined) {
            request.body["compute"] = params;
        }

        const requestId = await this.client.client.run(this.project, task, request);
        return requestId;
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
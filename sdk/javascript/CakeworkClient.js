import { CakeworkApiClient } from "@cakework/fern_client";

class CakeworkClient {
    constructor(project, token) {
        this.project = project;
        this.token = token;
        this.client = new CakeworkApiClient({
            project: project,
            xApiKey: token
        })
    }

    async run(task, params, compute) {        
        const request = {
            parameters: params,
            compute: compute
        };
        const requestId = await this.client.client.run(this.project, task, request);
        return requestId;
    }

    async getRunStatus(runId) {
        const runStatus = await this.client.client.getRunStatus(runId);
        return runStatus;
    }

    async getRunResult (runId) {
        const runResult = await this.client.client.getRunResult(runId);
        return runResult;
    }
}

export default CakeworkClient;
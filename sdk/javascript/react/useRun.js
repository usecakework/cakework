import { useState } from "react"
import CakeworkClient from "@cakework/client"

function useRun() {    
    const [runId, setRunId] = useState("");
    const [status, setStatus] = useState("");
    const [result, setResult] = useState(null);

    async function run(token, project, task, parameters, compute) {
        setRunId("");
        setRunId("");
        setResult(null);

        const cakework = new CakeworkClient(project, token);
        const runResponse = await cakework.run(task, parameters, compute);
        setRunId(runResponse.runId);
        var status;
        do {
            await new Promise(r => setTimeout(r, 1000));
            status = await cakework.getRunStatus(runResponse.runId);
            setStatus(status);
        } while (status !== "SUCCEEDED" && status !== "FAILED");

        const result = await cakework.getRunResult(runResponse.runId);
        setResult(result);
    }       

    return [runId, status, result, run];
}

export default useRun;
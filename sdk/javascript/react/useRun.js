import { useEffect, useState } from "react"
import CakeworkClient from "@cakework/client"

function useRun(token, project, task, parameters, compute) {
    const [status, setStatus] = useState("INITIALIZED");
    const [result, setResult] = useState(null);

    useEffect(() => {
        async function run() {
            const cakework = new CakeworkClient(project, token); 
            const runId = await cakework.run(task, parameters, compute);
            var status;
            do {
                await new Promise(r => setTimeout(r, 500));
                status = await cakework.getRunStatus(runId);
                setStatus(status);
            } while (status !== "SUCCEEDED" && status !== "FAILED");

            const result = await cakework.getRunResult(runId);
            setResult(result);
        }   

        run();
    })    

    return [status, result];
}

export default useRun;
import { useEffect, useState, useRef } from "react"
import CakeworkClient from "@cakework/client"

function useRun(token, project, task, parameters, compute) {
    const [status, setStatus] = useState("INITIALIZED");
    const [result, setResult] = useState(null);
    
    const didRun = useRef(false);

    useEffect(() => {
        if (didRun.current === true) {
            return
        }
        didRun.current = true;

        async function run() {
            const cakework = new CakeworkClient(project, token); 
            const runResponse = await cakework.run(task, parameters, compute);
            var status;
            do {
                await new Promise(r => setTimeout(r, 500));
                status = await cakework.getRunStatus(runResponse.runId);
                setStatus(status);
            } while (status !== "SUCCEEDED" && status !== "FAILED");

            const result = await cakework.getRunResult(runResponse.runId);
            setResult(result);
        }   

        run();
    }, [])    

    return [status, result];
}

export default useRun;
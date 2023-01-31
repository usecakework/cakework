import './App.css';
import useRun from '@cakework/client/react/useRun';

function App() {
  const [runId, status, result, run] = useRun();

  return (
    <div className="App">
      <div>This is an example of using a react hook to call Cakework.</div>
      <button onClick={() => run("YOUR_CLIENT_TOKEN_HERE", "react-example-backend", "say-hello", {"name":"jessie"})}>Start me!</button>
      <div>Your Run ID: {runId}</div>
      <div>Run Status: {status}</div>
      <div>Run Result: {result}</div>
    </div>
  );
}

export default App;

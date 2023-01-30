import './App.css';
import useRun from '@cakework/client/react/useRun';

function App() {
  const [status, result] = useRun("264d23638f85a2b344595b91e04735d6c63a59bd9aaf9ae9505585b98c1af030", "react-example-backend", "say-hello", {"name":"jessie"});

  return (
    <div className="App">
      <div>This is an example of using a react hook to call Cakework.</div>
      <div>{status}</div>
      <div>{result}</div>
    </div>
  );
}

export default App;

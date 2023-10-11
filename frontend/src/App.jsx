
import './App.css'

function App() {
  const sendRequest = async () =>{
   const res = await fetch("/test", {
    method: "GET"
   })

   let output = await res.json()
   console.log(output.message)
  }

  return (
    <div className='text-3xl'>
      <button onClick={sendRequest}>Send Request! :O</button>
    </div>
  )
}

export default App

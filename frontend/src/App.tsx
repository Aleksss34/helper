
import './App.css'
import {SearchStream} from "./components/Search.tsx";
import {ParseLegistationButton} from "./components/Parser-legistation.tsx";
import {ParseButton} from "./components/Parser.tsx";


function App() {


  return (
      <>
      <SearchStream></SearchStream>
          <ParseLegistationButton></ParseLegistationButton>

      <br/>
          <ParseButton></ParseButton>
      </>
  )
}

export default App

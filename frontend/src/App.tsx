
 // Убедись, что тут PNG с прозрачным фоном
import { SearchStream } from './components/Search.tsx';
import {UserProvider} from "./components/GetUser.tsx";
import {BigSmouk} from "./components/BigSmouk.tsx";
 import {AuthProvider} from "./components/AuthModal.tsx";


function App() {
    return (
        <>



            <BigSmouk />
            <AuthProvider>
                <UserProvider>
                    <SearchStream />
                </UserProvider>
            </AuthProvider>

           

        </>
    );

}





export default App;
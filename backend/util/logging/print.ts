import { SECRETS } from "../../config/env"

const log = (input: any) =>{
    let testMode: string = SECRETS.testMode as string
    if(testMode == "false"){
        console.log(input)
    }
}

export default log
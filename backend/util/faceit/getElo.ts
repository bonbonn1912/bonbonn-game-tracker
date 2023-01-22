import { SECRETS } from "../../config/env"

import faceitPlayerReponse from "../../types/player"


const getFaceitPlayer = async (username: string) : Promise<faceitPlayerReponse | null> =>{
    let url: string = SECRETS.faceit.baseUrl + `/players?nickname=${username}`
   
    let responseRaw: Response = await fetch(url, {
            method: "GET", 
            headers: SECRETS.faceit.header
    })
    if(responseRaw.status === 200){
        let player: faceitPlayerReponse = await responseRaw.json()
        return player; 
    }else{
        return null;        
    }
}

export { getFaceitPlayer }
import { SECRETS } from "../../config/env"

import faceitPlayerReponse from "../../@types/player"


const getFaceitPlayer = async (username: string) : Promise<faceitPlayerReponse> =>{
    let url: string = SECRETS.faceit.baseUrl + `/players?nickname=${username}`
   
    let responseRaw: Response = await fetch(url, {
            method: "GET", 
            headers: SECRETS.faceit.header
    })
        let player: faceitPlayerReponse = await responseRaw.json()
    return player; 
}

export { getFaceitPlayer }
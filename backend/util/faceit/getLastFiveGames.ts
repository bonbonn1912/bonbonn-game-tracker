import { SECRETS } from "../../config/env"

import faceitPlayerReponse, {faceitMatchHistory} from "../../@types/player"


const getLastFiveGames = async (faceitId: string) : Promise<string[]> =>{
    
    let url: string =  SECRETS.faceit.baseUrl + `/players/${faceitId}/stats/csgo`
    console.log(url)
    let responseRaw: Response = await fetch(url, {
            method: "GET", 
            headers: SECRETS.faceit.header
    })
    let lastFive: faceitMatchHistory = await responseRaw.json()
    return lastFive.lifetime["Recent Results"]; 
    
}


export { getLastFiveGames }
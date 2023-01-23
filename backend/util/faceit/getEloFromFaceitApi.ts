import { SECRETS } from "../../config/env"

import faceitPlayerReponse from "../../@types/player"


const getFaceitPlayer = async (username: string, id?: string) : Promise<faceitPlayerReponse> =>{
    
    let path: string = id == undefined ? `/players?nickname=${username}` : `${isFaceitID(id) ? `/players/${id}` : `/players?game=csgo&game_player_id=${id}`}`
    let url: string =  SECRETS.faceit.baseUrl + path
    console.log(url)
    let responseRaw: Response = await fetch(url, {
            method: "GET", 
            headers: SECRETS.faceit.header
    })
        let player: faceitPlayerReponse = await responseRaw.json()
    return player; 
}

const isFaceitID = (id: string): boolean =>{
   return new RegExp(SECRETS.regex.faceitId).test(id)
}

export { getFaceitPlayer }
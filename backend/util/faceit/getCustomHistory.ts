import { SECRETS } from "../../config/env"
import { matchHistory } from "../../@types/customHistory"


const getCustomFaceitHistory = async (faceitId: string, limit: number): Promise<matchHistory> =>{
    
    let url: string =  SECRETS.faceit.baseUrl + `/players/${faceitId}/history?game=csgo&offset=0&limit=${limit}`
    console.log(url)
    let responseRaw: Response = await fetch(url, {
            method: "GET", 
            headers: SECRETS.faceit.header
    })
    let customMatchHistory: matchHistory = await responseRaw.json()
    return customMatchHistory
    
}


export { getCustomFaceitHistory }
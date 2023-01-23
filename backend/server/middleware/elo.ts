import { Request, Response, NextFunction} from "express"

import faceitPlayerReponse from "../../@types/player"
import faceitElo from "../../@types/level"
import { getFaceitPlayer } from "../../util/faceit/getEloFromFaceitApi"
import { faceitRankInformation } from "../../util/faceit/getEloInformation"
import { eloReponse } from "../../util/faceit/createResponseString"

const getFaceitElo = async (req: Request, res: Response, next: NextFunction) =>{
    
    try{
        let player: faceitPlayerReponse = await getFaceitPlayer(req.query.username as string, req.query.id as string)
        let rankInformation: faceitElo = faceitRankInformation(player)
        rankInformation.responseString = eloReponse(rankInformation)
        player.local = rankInformation
        req.player = player
        next()
    }catch{
        res.status(404).send("Invalid username / steam64 id / faceit id")
    }   
}

export { getFaceitElo}
   

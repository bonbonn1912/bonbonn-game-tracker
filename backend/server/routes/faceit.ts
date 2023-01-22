import { Router, Request, Response} from "express";
import { getFaceitPlayer } from "../../util/faceit/getElo";

import faceitPlayerReponse from "../../types/player";
import { faceitRankInformation } from "../../util/faceit/level";
import { validUsername } from "../../util/validParameter";

const faceitRouter: Router = Router();

faceitRouter.get("/elo", async (req: Request, res: Response) => {
    let username: any = req.query.username
    if(username != undefined && validUsername(username)){
        let player: faceitPlayerReponse | null = await getFaceitPlayer(username)
        if(player != null){
            let rankInformation = faceitRankInformation(player.games.csgo.faceit_elo)
            res.status(200).json(rankInformation)
        }else{
            res.status(404).json("Invalid username")
        }
    }else{
        res.status(404).json({"error": "Nicknames can be made of letters (a-z, A-Z), numbers (0-9), '_' and '-' only. 3-12 characters"})
    }
    
})


export default faceitRouter
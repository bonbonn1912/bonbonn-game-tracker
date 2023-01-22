import { Router, Request, Response} from "express";
import { getFaceitPlayer } from "../../util/faceit/getElo";

import faceitPlayerReponse from "../../types/player";
import { faceitRankInformation } from "../../util/faceit/level";
import { validUsername } from "../../util/validParameter";
import { eloReponse } from "../../util/faceit/response";

const faceitRouter: Router = Router();

faceitRouter.get("/elo", async (req: Request, res: Response) => {
    let username: any = req.query.username

    if(username == undefined || !validUsername(username)){
        res.status(404).send("Nicknames can be made of letters (a-z, A-Z), numbers (0-9), '_' and '-' only. 3-12 characters")
        return;
    }

    try{
        let player: faceitPlayerReponse = await getFaceitPlayer(username)
        let rankInformation = faceitRankInformation(player.games.csgo.faceit_elo)
        let responseString = eloReponse(username,rankInformation)
        res.status(200).send(responseString)
    }catch{
        res.status(404).send("Invalid username")
    }   

})


export default faceitRouter
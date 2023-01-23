import { Request, Response, NextFunction} from "express"
import { validFaceitUsername, validSteam64Id, validFaceitId } from "../../util/validRequestParameter"
import { SECRETS } from "../../config/env"


const checkEloInput = (req: Request, res: Response, next: NextFunction) =>{
    
    let username: any = req.query.username

    if(username == undefined || !validFaceitUsername(username)){
        res.status(404).send("Nicknames can be made of letters (a-z, A-Z), numbers (0-9), '_' and '-' only. 3-12 characters")
        return;
    }
    next();
}

const checkEloBySteamIdInput = (req: Request, res: Response, next: NextFunction) =>{
    let steam64id: any = req.query.id
    if(steam64id == undefined || !validSteam64Id(steam64id)){
        res.status(404).send("Steam 64 Ids are made of 17 numbers (0-9)")
        return;
    }
    next();
}

const checkEloByFaceitIdInput = (req: Request, res: Response, next: NextFunction) =>{
    let faceitId: any = req.query.id
    if(faceitId == undefined || !validFaceitId(faceitId)){
        res.status(404).send("Faceit Ids must match following pattern : " + SECRETS.regex.faceitId)
        return;
    }
    next();
}

export { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput }
import { Request, Response, NextFunction} from "express"
import { validUsername } from "../../util/validRequestParameter"


const checkEloInput = (req: Request, res: Response, next: NextFunction) =>{
    
    let username: any = req.query.username

    if(username == undefined || !validUsername(username)){
        res.status(404).send("Nicknames can be made of letters (a-z, A-Z), numbers (0-9), '_' and '-' only. 3-12 characters")
        return;
    }
    next();
}

export { checkEloInput }
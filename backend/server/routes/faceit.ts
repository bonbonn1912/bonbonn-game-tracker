import { Router, Request, Response} from "express";



import { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput } from "../middleware/checkInput";
import { getFaceitElo } from "../middleware/elo";


const faceitRouter: Router = Router();

faceitRouter.get("/elo", checkEloInput,getFaceitElo,async (req: Request, res: Response) => {
    
    res.send(req.player.local.responseString)

})

faceitRouter.get("/elobysteamid", checkEloBySteamIdInput,getFaceitElo,async (req: Request, res: Response) => {
    
    res.send(req.player.local.responseString)
})

faceitRouter.get("/elobyid", checkEloByFaceitIdInput,getFaceitElo,async (req: Request, res: Response) => {
    
    res.send(req.player.local.responseString)
    
})


export default faceitRouter
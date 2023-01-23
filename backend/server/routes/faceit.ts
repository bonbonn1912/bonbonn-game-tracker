import { Router, Request, Response} from "express";



import { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput } from "../middleware/checkInput";
import { getFaceitElo, getLastFive } from "../middleware/elo";
import { addPlayerToDB } from "../../util/database/addToDatabase";
import { InsertType } from "../../@types/insertTypes";

const faceitRouter: Router = Router();

faceitRouter.get("/elo", checkEloInput,getFaceitElo,async (req: Request, res: Response) => {
    addPlayerToDB(req.player, InsertType.ELO)
    res.send(req.player.local.responseString)

})

faceitRouter.get("/elobyid", checkEloByFaceitIdInput,getFaceitElo,async (req: Request, res: Response) => {
    addPlayerToDB(req.player, InsertType.ELOBYID)
    res.send(req.player.local.responseString)
    
})

faceitRouter.get("/elobysteamid", checkEloBySteamIdInput,getFaceitElo,async (req: Request, res: Response) => {

    addPlayerToDB(req.player, InsertType.ELOBYSTEAMID)
    res.send(req.player.local.responseString)
})



faceitRouter.get("/checkelo", checkEloInput, getFaceitElo, getLastFive,async (req: Request, res: Response) => {
    addPlayerToDB(req.player, InsertType.CHECKELO)
    res.send(req.player.local.responseString)
 
})


export default faceitRouter
import { Router, Request, Response} from "express";



import { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput, checkMatchHistoryInput } from "../middleware/checkInput";
import { getCustomHistory, getFaceitElo, getLastFive } from "../middleware/elo";
import { addPlayerToDB } from "../../util/database/addToDatabase";
import { InsertType } from "../../@types/insertTypes";
import { getCustomFaceitHistory } from "../../util/faceit/getCustomHistory";

const faceitRouter: Router = Router();

faceitRouter.get("/elo", checkEloInput,getFaceitElo, (req: Request, res: Response) => {
    addPlayerToDB(req.player, InsertType.ELO)
    res.send(req.player.local.responseString)

})

faceitRouter.get("/elobyid", checkEloByFaceitIdInput,getFaceitElo, (req: Request, res: Response) => {
    addPlayerToDB(req.player, InsertType.ELOBYID)
    res.send(req.player.local.responseString)
    
})

faceitRouter.get("/elobysteamid", checkEloBySteamIdInput,getFaceitElo,(req: Request, res: Response) => {

    addPlayerToDB(req.player, InsertType.ELOBYSTEAMID)
    res.send(req.player.local.responseString)
})



faceitRouter.get("/checkelo", checkEloInput, getFaceitElo, getLastFive, (req: Request, res: Response) => {
    addPlayerToDB(req.player, InsertType.CHECKELO)
    res.send(req.player.local.responseString)
 
})

faceitRouter.get("/matchhistory", checkMatchHistoryInput,getFaceitElo, getCustomHistory, (req: Request, res: Response) => {
   // addPlayerToDB(req.player, InsertType.CHECKELO)
   res.send(req.player.local.responseString)
})


export default faceitRouter
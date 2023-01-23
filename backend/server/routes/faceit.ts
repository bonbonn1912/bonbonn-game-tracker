import { Router, Request, Response} from "express";



import { checkEloInput } from "../middleware/checkInput";
import { getFaceitElo } from "../middleware/elo";


const faceitRouter: Router = Router();

faceitRouter.get("/elo", checkEloInput,getFaceitElo,async (req: Request, res: Response) => {
    
    res.send(req.responseString)
    
})


export default faceitRouter
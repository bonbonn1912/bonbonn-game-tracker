import { Request, Response, Router } from "express";
import { getLive } from "../../../util/liveGames";


const gtGetRouter = Router()


gtGetRouter.get("/allgames", (req: Request, res: Response) =>{
    console.log("ALl Games Endpoint hit")
    res.json(Object.fromEntries(getLive()))
})

export { gtGetRouter } 
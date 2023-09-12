import { SECRETS } from "../../../config/env";
import matchroom from "../../../@types/matchroom";
import log from "../../logging/print";

const getMap = async (matchId: string) : Promise<string> => {
    log("Get Map for match id : " + matchId)
    const url: string = SECRETS.faceit.baseUrl + `/matches/${matchId}`
    const responseRaw: Response = await fetch(url, {
        method: 'GET',
        headers: SECRETS.faceit.header
    })
    const matchInfo: matchroom = await responseRaw.json()

    return matchInfo.voting.map.pick[0]
}

export { getMap }
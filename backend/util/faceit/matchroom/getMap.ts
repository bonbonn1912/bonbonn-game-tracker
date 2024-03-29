import { SECRETS } from "../../../env";
import matchroom from "../../../@types/matchroom";

const getMap = async (matchId: string) : Promise<string> => {
    console.log("Get Map for match id : " + matchId)
    const url: string = SECRETS.faceit.baseUrl + `/matches/${matchId}`
    const responseRaw: Response = await fetch(url, {
        method: 'GET',
        headers: SECRETS.faceit.header
    })
    const matchInfo: matchroom = await responseRaw.json()

    return matchInfo.voting.map.pick[0]
}

export { getMap }
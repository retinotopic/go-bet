<script lang="ts">
import { page  } from '$app/stores';
import { onMount } from 'svelte';
import Card from '../../components/card.svelte';
import Hand from '../../components/hand.svelte';


interface Ctrl {
	Place     : number      
	CtrlInt  :  number     
	CtrlString :string   
}
interface GameBoard {
	Cards:       string[] 
	Bank  :      number  
	TurnPlace:   number   
	DealerPlace :number           
	Deadline   : number 
	Blind      : number           
	Active     : boolean 
	IsRating   : boolean   
}
interface PlayUnit {
	Cards   :  string[] 
	IsFold  :  boolean  
	IsAway  :  boolean 
	HasActed:  boolean
	Bank   :   number       
	Bet   :    number      
	Place :    number        
	TimeTurn : number           
	Name    :  string    
	UserId :  string      
}

function toGameBoard(data: any): GameBoard | null {
    if (!(typeof data.Active === 'string')) {
        return null;
    }
    const gameBoard: GameBoard = {
        Cards: Array.isArray(data.Cards) ? data.Cards : [],
        Bank: !isNaN(+data.Bank) ? +data.Bank : 0,
        TurnPlace: !isNaN(+data.TurnPlace) ? +data.TurnPlace : 0,
        DealerPlace: !isNaN(+data.DealerPlace) ? +data.DealerPlace : 0,
        Deadline: !isNaN(+data.Deadline) ? +data.Deadline : 0,
        Blind: !isNaN(+data.Blind) ? +data.Blind : 0,
        Active: typeof data.Active === 'string' ? data.Active.toLowerCase() === 'true' : false,
        IsRating: typeof data.IsRating === 'string' ? data.IsRating.toLowerCase() === 'true' : false
    }
    return gameBoard;
}
function toPlayUnit(data: any): PlayUnit | null {
    if (!(typeof data.UserId === 'string')) {
        return null;
    }
    const playUnit: PlayUnit = {
    Cards: Array.isArray(data.Cards) && data.Cards.length === 3 ? data.Cards : ['', '', ''],

    IsFold: typeof data.IsFold === 'string' ? data.IsFold.toLowerCase() === 'true' : false,
    IsAway: typeof data.IsAway === 'string' ? data.IsAway.toLowerCase() === 'true' : false,
    HasActed: typeof data.HasActed === 'string' ? data.HasActed.toLowerCase() === 'true' : false,

    Bank: !isNaN(+data.Bank) ? +data.Bank : 0,
    Bet: !isNaN(+data.Bet) ? +data.Bet : 0,
    Place: !isNaN(+data.Place) ? +data.Place : 0,
    TimeTurn: !isNaN(+data.TimeTurn) ? +data.TimeTurn : 0,

    Name: data.Name?.toString() ?? 'Unknown Player',
    UserId: data.UserId
    }
    return playUnit
}
let room_id = $state($page.params);
let gameBoard =  $state<GameBoard>();
let Places = $state<PlayUnit[]>(Array(8).fill({} as PlayUnit));
let myPlace = $state<PlayUnit>();
let timer = $state(0);
let user_id = $state("");
let precall = $state(false);
let precheckfold = $state(false);

let socket: WebSocket;

onMount(() => {
    connectWebSocket();
});

function sendmsg(msg :string) {
    socket.send(msg)
};
function connectWebSocket() {
    socket = new WebSocket('ws://localhost:8080/lobby'+room_id);

    socket.onclose = () => {

    };

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (!data) {
            return
        }
        if (data.user_id) {
            user_id = data.user_id
        }
        let pl = toPlayUnit(data)
        if (pl) {
            Places[pl.Place] = pl
            if (user_id === pl.UserId) {
                myPlace = Places[pl.Place]
            }
        }

        let gb = toGameBoard(data);
        if (gb) {
            gameBoard = gb;
        }
    };
}
</script>

{#each Places as _, i}
    <div style="player">
        <Hand cards={Places[i].Cards}/>
    </div>
{/each}
{#if myPlace?.Place === gameBoard?.TurnPlace }
{:else}
<input type="checkbox" bind:checked={precall} name=""/>
Call
<input type="checkbox" bind:checked={precheckfold} />
Check/fold
{/if}
{#if myPlace?.Place === gameBoard?.TurnPlace }
//timer
{/if}
<div>{gameBoard}</div>
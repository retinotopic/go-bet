<script lang="ts">
import { page  } from '$app/stores';
import { onMount } from 'svelte';
import Card from '../../components/card.svelte';
import Hand from '../../components/hand.svelte';
import Timer from '../../components/timer.svelte';

interface Ctrl {
	Place : number      
	Ctrl : number     
	Text : string   
}
interface GameBoard {
	Cards:       string[] 
	Bank  :      number
	MaxBet:      number  
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
	UserId :   string      
}
function InitPlayers(count:number): PlayUnit[] {
	let arr: PlayUnit[] =[];
	for (var i = 0; i < count; i++) {
		arr.push(toPlayUnit({}))
	} 
	return arr
}
function toCtrl(data: any): Ctrl {
    const ctrl: Ctrl = {
        Place: !isNaN(+data?.Place) ? +data.Place : 0,
        Ctrl: !isNaN(+data?.Ctrl) ? +data.Ctrl : 0,
        Text: data?.Text?.toString() ?? "",
    }
    return ctrl;
}
//data.Cards.every(item => typeof item === 'string'
function toGameBoard(data: any): GameBoard  {
    const gameBoard: GameBoard = {
        Cards: Array.isArray(data?.Cards) && data.Cards ? data.Cards : [],
        Bank: !isNaN(+data?.Bank) ? +data.Bank : 0,
        MaxBet: !isNaN(+data?.MaxBet) ? +data.MaxBet : 0,

        TurnPlace: !isNaN(+data?.TurnPlace) ? +data.TurnPlace : 0,
        DealerPlace: !isNaN(+data?.DealerPlace) ? +data.DealerPlace : 0,
        Deadline: !isNaN(+data?.Deadline) ? +data.Deadline : 0,
        Blind: !isNaN(+data?.Blind) ? +data.Blind : 0,
        //Active: typeof data.Active === 'string' ? data.Active.toLowerCase() === 'true' : false,
        Active: typeof data?.Active === 'boolean' ? data.Active : false,
        //IsRating: typeof data.IsRating === 'string' ? data.IsRating.toLowerCase() === 'true' : false
        IsRating: typeof data?.IsRating === 'boolean' ? data.IsRating : false
    }
    return gameBoard;
}
function toPlayUnit(data: any): PlayUnit  {
    const playUnit: PlayUnit = {
        Cards: Array.isArray(data?.Cards) && data.Cards.length === 3 ? data.Cards : ['', '', ''],

        //IsFold: typeof data.IsFold === 'string' ? data.IsFold.toLowerCase() === 'true' : false,
        IsFold: typeof data?.IsFold === 'boolean' ? data.IsFold : false,    
        
        //IsAway: typeof data.IsAway === 'string' ? data.IsAway.toLowerCase() === 'true' : false,
        IsAway: typeof data?.IsAway === 'boolean' ? data.IsAway : false,

        // HasActed: typeof data.HasActed === 'string' ? data.HasActed.toLowerCase() === 'true' : false,
        HasActed: typeof data?.HasActed === 'boolean' ? data.HacActed : false,

        Bank: !isNaN(+data?.Bank) ? +data.Bank : 0,
        Bet: !isNaN(+data?.Bet) ? +data.Bet : 0,
        Place: !isNaN(+data?.Place) ? +data.Place : 0,
        TimeTurn: !isNaN(+data?.TimeTurn) ? +data.TimeTurn : 0,

        Name: data?.Name?.toString() ?? 'Unknown Player',
        UserId: data?.Name?.toString() ?? 'undefined id'
    }
    return playUnit
}
let user_id = $state("");
let room_id = $state($page.params);
let gameBoard =  $state<GameBoard>(toGameBoard({}));
let ctrl =  $state<Ctrl>(toCtrl({}));
let places = $state<PlayUnit[]>(InitPlayers(8));
let myPlaceId = $state(0);

let required = $state(0);
let hideSlider = $state(false);
let timer = $state(0);
let hasacted = $state(false)

let socket: WebSocket;

onMount(() => {
    connectWebSocket();
});

function sendmsg(c: Ctrl) {
    socket.send(JSON.stringify(c))
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
        
        places[pl.Place] = pl
        if (user_id === pl.UserId) {    
            myPlaceId = pl.Place;
        }        

        gameBoard = toGameBoard(data);
        timer = gameBoard.Deadline

        
        if (myPlaceId === gameBoard.TurnPlace) {
            hasacted = false
            required = gameBoard.MaxBet - places[myPlaceId].Bet
            if (required > places[myPlaceId].Bank) {
                hideSlider = true
                ctrl.Ctrl = places[myPlaceId].Bank
            } else {
                hideSlider = false
                ctrl.Ctrl = required
            }
        }    
        
    };
}
</script>

{#each places as _, i}
    <div style="player">
        <Hand cards={places[i].Cards}/>
        {#if places[i].Place === gameBoard?.TurnPlace }
            <Timer deadline={timer}/>
        {/if}        
    </div>
{/each}

{#each gameBoard.Cards as _,i}
    <Card ranksuit={gameBoard.Cards[i]} />
{/each}

{#if myPlaceId === gameBoard?.TurnPlace && !hasacted }
    <button onclick={ () => {
        sendmsg(ctrl);
        hasacted = true; }}>  
    	{#if ctrl.Ctrl == places[myPlaceId].Bank}
    		ALL IN
    	{:else if ctrl.Ctrl > required}
    		RAISE
    	{:else if ctrl.Ctrl == 0}
    		CHECK
    	{:else}
    		CALL
    	{/if}
    </button>
    <button onclick={ () => {
        ctrl.Ctrl = 0;
        sendmsg(ctrl);
        hasacted = true; }}>  
    		FOLD
    </button>
    <input type="range" hidden={hideSlider}  bind:value={ctrl.Ctrl}
        min={required} max={places[myPlaceId].Bank}>
    {ctrl}
{/if}

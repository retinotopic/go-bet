<script lang="ts">
import { page  } from '$app/stores'
import { onMount } from 'svelte'
import Card from '../../components/card.svelte'
import Hand from '../../components/hand.svelte'
import Timer from '../../components/timer.svelte'

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
	DealerPlace: number
	AdminPlace:  number           
	Deadline  :  number 
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
	let arr: PlayUnit[] =[]
	for (var i = 0; i < count; i++) {
		arr.push(toPlayUnit({}))
	} 
	return arr
}

function toCtrl(data: any): Ctrl {
    const ctrl: Ctrl = {
        Place: !isNaN(+data?.Place) ? +data.Place : -2,
        Ctrl: !isNaN(+data?.Ctrl) ? +data.Ctrl : 0,
        Text: data?.Text?.toString() ?? "",
    }
    return ctrl
}

//data.Cards.every(item => typeof item === 'string'
function toGameBoard(data: any): GameBoard  {
    let gameBoard: GameBoard = {
        Cards: Array.isArray(data?.Cards) && data.Cards ? data.Cards : [],
        Bank: !isNaN(+data?.Bank) ? +data.Bank : 0,
        MaxBet: !isNaN(+data?.MaxBet) ? +data.MaxBet : 0,

        TurnPlace: !isNaN(+data?.TurnPlace) ? +data.TurnPlace : 0,
        AdminPlace: !isNaN(+data?.AdminPlace) ? +data.AdminPlace : 0,
        DealerPlace: !isNaN(+data?.DealerPlace) ? +data.DealerPlace : 0,
        Deadline: !isNaN(+data?.Deadline) ? +data.Deadline : 0,
        Blind: !isNaN(+data?.Blind) ? +data.Blind : 0,
        Active: typeof data?.Active === 'boolean' ? data.Active : false,
        IsRating: typeof data?.IsRating === 'boolean' ? data.IsRating : false
    }
    return gameBoard
}
function toPlayUnit(data: any): PlayUnit  {
    let playUnit: PlayUnit = {
        Cards: Array.isArray(data?.Cards) && data.Cards.length === 3 ? data.Cards : ['', '', ''],
        IsFold: typeof data?.IsFold === 'boolean' ? data.IsFold : false,    
        IsAway: typeof data?.IsAway === 'boolean' ? data.IsAway : false,
        HasActed: typeof data?.HasActed === 'boolean' ? data.HacActed : false,
        Bank: !isNaN(+data?.Bank) ? +data.Bank : 0,
        Bet: !isNaN(+data?.Bet) ? +data.Bet : 0,
        Place: !isNaN(+data?.Place) ? +data.Place : 0,
        TimeTurn: !isNaN(+data?.TimeTurn) ? +data.TimeTurn : 0,
        Name: data?.Name?.toString() ?? 'Unknown Player',
        UserId: data?.UserId?.toString() ?? 'undefined id'
    }
    return playUnit
}
let userId = $state("")
let roomId = $state($page.params)
let gameBoard =  $state<GameBoard>(toGameBoard({}))

let ctrl =  $state<Ctrl>(toCtrl({})) // OUTPUT
let ctrlSeat =  $state<Ctrl>(toCtrl({})) // OUTPUT
let ctrlMsg =  $state<Ctrl>(toCtrl({})) // OUTPUT
let ctrlStart =  $state<Ctrl>(toCtrl({})) // OUTPUT

let msgs =  $state<Ctrl[]>([]) //INPUT

let places = $state<PlayUnit[]>(InitPlayers(8))
let myPlaceId = $state(0)

let required = $state(0)
let hideSlider = $state(false)
let timer = $state(0)
let hasacted = $state(false)

let takeSeatForm = $state(false)
let socket: WebSocket

onMount(() => {
    connectWebSocket()
})

function sendmsg(c: Ctrl) {
    socket.send(JSON.stringify(c))
}
function connectWebSocket() {
    socket = new WebSocket('ws://localhost:8080/lobby'+roomId)

    socket.onclose = () => {

    }

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data)
        if (!data) {
            return
        }
        if (data.userId) {
            userId = data.userId
        }
        
        switch(data) { 
            case data?.UserId?.toString() ? true : false: { 
                let pl = toPlayUnit(data)
                places[pl.Place] = pl
                if (userId === pl.UserId) {    
                    myPlaceId = pl.Place
                    ctrlMsg.Place = pl.Place
                }        
 
            } 
            case typeof data?.Active === 'boolean' ? data.Active : false: { 
                gameBoard = toGameBoard(data)
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
             } 
            default: {
                let ctrlmsg = toCtrl(data)
                msgs.push(ctrlmsg)
            } 
        } 
      
    }
}
</script>

// messages
{#each msgs as _, i}
    {places[msgs[i].Place].Name}: {msgs[i].Text}
{/each}
<input bind:value={ctrlMsg.Text}>
    ENTER TEXT
<button onclick={ () => {
    sendmsg(ctrlMsg)
    ctrlMsg.Text = "" }}>
    SEND MESSAGE
</button>

//game board cards
{#each gameBoard.Cards as _,i}
    <Card ranksuit={gameBoard.Cards[i]} />
{/each}

// player's turn management
{#if myPlaceId === gameBoard?.TurnPlace && !hasacted }
    <button onclick={ () => {
        sendmsg(ctrl)
        hasacted = true }}>  
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
        ctrl.Ctrl = 0
        sendmsg(ctrl)
        hasacted = true }}>  
    		FOLD
    </button>
    <input type="range" hidden={hideSlider}  bind:value={ctrl.Ctrl}
        min={required} max={places[myPlaceId].Bank}>
    {ctrl}
{/if}

// each player place at the board
{#each places as _, i}
    <div style="player">
        <Hand cards={places[i].Cards}/>
        {#if places[i].Place === gameBoard?.TurnPlace }
            <Timer deadline={timer}/>
        {/if}        
        {#if !gameBoard.Active} // taking up a seat for a custom game
            <input type="checkbox" bind:value={takeSeatForm}>
            {#if takeSeatForm}
                <input bind:value={ctrlSeat.Text}>
                    ENTER NAME
                
                <button onclick={ () => {
                    takeSeatForm = false
                    ctrlSeat.Text = userId + ctrlSeat.Text
                    ctrlSeat.Ctrl = userId.length
                    ctrlSeat.Place = i
                    sendmsg(ctrlSeat)
                    ctrlSeat.Text = ""
                    }}>  
                		TAKE SEAT
                </button>
            {/if}
            // tweaking custom game if player an admin
            {#if myPlaceId === gameBoard.AdminPlace && !gameBoard.Active }
                Blind Level Round Length (In Seconds)
                default is 300 seconds (5 min)
                <input bind:value={ctrlStart.Text}>
                Initial Stack for Players
                <input bind:value={ctrlSeat.Ctrl}> 
                <button onclick={ () => {
                    if (ctrlStart.Text === "") {
                        ctrlStart.Text = "300"
                    } 
                    sendmsg(ctrlStart)
                    ctrlStart.Text = ""
                    ctrlStart.Ctrl = 0
                    }}>  
                		START GAME
                </button>
                
            {/if}
        {/if}        
    </div>
{/each}

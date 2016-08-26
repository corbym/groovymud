package commands.player

/* Copyright 2008 Matthew Corby-Eaglen
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. 
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0 
 *
 * Unless required by applicable law or agreed to in writing, software 
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. 
 * See the License for the specific language governing permissions and 
 * limitations under the License. 
 */
import org.groovymud.object.room.*
import org.groovymud.engine.event.messages.MessageEvent
import org.groovymud.engine.event.EventScope

def thisRoom = source.getCurrentContainer();

def exit = thisRoom.getExit(args[0]);
if(exit != null){
	exit.go(source)	
}else{
	source.getTerminalOutput().writeln "You cannot go ${args[0]}.";
}
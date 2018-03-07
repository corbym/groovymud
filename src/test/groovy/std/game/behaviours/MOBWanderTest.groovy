/**
 * 
 */
package std.game.behaviours

import std.game.objects.alive.MOBImpl

/**
 * @author corbym
 *
 */

public class MOBWanderTest extends GroovyTestCase{
	
	/* (non-Javadoc)
	 * @see std.game.behaviours.MOBWander#doBehaviour(org.groovymud.object.MudObject)
	 */
	final void testDoBehaviour(){
		def config = new GroovyDice()
		config.numberGenerator = { sides -> return 1 }
		config.initialize()
		
		def exit1 = [name : {"west" },
		getDirection : {"west"}]as Exit
		def exit2 = [name : {"east"},
		getDirection : {"east"}] as Exit

		
		def mockContainer = [getExits: {[exit1]}] as RoomImpl
		def mob =[getCurrentContainer: {mockContainer}
		, getInventoryHandler : {mockHandler},
			go : { args -> 
					assertTrue args == "west"
			}
		] as MOBImpl
		
		
		MOBWander wander = new MOBWander()
		wander.doBehaviour(mob)
		
	}
	
}
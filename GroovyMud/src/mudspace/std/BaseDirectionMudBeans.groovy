package std
/**

beans{
	
	
	'std:north'(ExitImpl, name:"north", shortNames:["n"]){ bean ->
		bean.'abstract' = true		
		direction = "north"
		arrivalDirection = "south"		
	}
	'std:south'(ExitImpl, name:"south", shortNames:["s"]){ bean ->
		bean.'abstract' = true
		direction = "south"
		arrivalDirection = "north"		
	}
	'std:east'(ExitImpl,  name:"east", shortNames:["e"]){ bean ->
		bean.'abstract' = true
		direction = "east"
		arrivalDirection = "west"		
	}
	'std:west'(ExitImpl,  name:"west", shortNames:["w"]){ bean ->
		bean.'abstract' = true
		direction = "west"
		arrivalDirection = "east"
	}
	'std:northeast'(ExitImpl, name:"northeast", shortNames:["ne"]){ bean ->
		direction = "northeast"
		arrivalDirection = "southwest"
	}
	'std:southeast'(ExitImpl, name:"southeast", shortNames:["se"]){ bean ->
		bean.'abstract' = true
		arrivalDirection = "northwest"
		shortNames = ["se"]
	}
	'std:northwest'(ExitImpl, name:"northwest", shortNames:["nw"]){ bean ->
		direction = "northwest"		
	}
	'std:southwest'(ExitImpl, name:"southwest", shortNames:["sw"]){ bean ->
		direction = "southwest"
		arrivalDirection = "northeast"
		shortNames = ["sw"]
	}
	'std:up'(ExitImpl, name:"up", shortNames:["u"]){ bean ->
		
		direction = "up"
		arrivalDirection = "below"
	}
	'std:down'(ExitImpl,  name:"down", shortNames:["d"]){ bean ->
		direction = "down"
		arrivalDirection = "above"		
	}
	
	'std:in'(ExitImpl, name:"in", shortNames:["in", "inside"]){ bean ->
		articleRequired = false
		direction = "in"
		arrivalDirection = "outside"
	}
	'std:out'(ExitImpl, name:"out", shortNames:["out", "outside"]){ bean ->
		articleRequired = false 
		direction = "out"
		arrivalDirection = "inside"	
	}
}
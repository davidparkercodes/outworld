package outworld

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

var homonymProseMasks = []struct{ seed, name, in string }{
	{"Ruby Kelly called about the invoice.", "Ruby", "Use Ruby.run(1) here."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Use Ruby::Version here."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Call Config.Ruby now."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Module.Ruby is loaded."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Open lib/Ruby/core.rb please."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Look in /usr/lib/Ruby/core for it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "See https://example.com/Ruby/docs today."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Load ./Ruby/setup now."},
	{"Crystal Kelly called about the invoice.", "Crystal", "System.Crystal fails."},
	{"Ruby Kelly called about the invoice.", "Ruby", "``Ruby'' called and ``Anna'' replied."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby' called and `Tom' replied."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I took James` car, then Ruby called, then I fed Chris` dog."},
	{"Ruby Kelly called about the invoice.", "Ruby", "In the 90` Ruby worked there, in the 2000` she left."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Run `x` then Ruby called about the invoice."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Run `x then Ruby called about the invoice.\n\nOK` done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Run ``x` then Ruby called about the invoice `` done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Run ```x``` then Ruby called about the invoice."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby`s car broke and Ruby called about it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby` called about the invoice ` again."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby Kelly called about the invoice`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`ruby kelly called about the invoice`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby called me about the invoice yesterday`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby` and `Kelly` called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby Kelly`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Kelly, Ruby`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ask `Ruby` (Kelly) about it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby`'s car broke."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Notes ` Ruby called about the invoice ` end"},
	{"Ruby Kelly called about the invoice.", "Ruby", "a `\nRuby called about the invoice.\n` b"},
	{"Ruby Kelly called about the invoice.", "Ruby", "a `\n \nRuby called about the invoice.\n` b"},
	{"Ruby Kelly called about the invoice.", "Ruby", "x`` y `` Ruby called about the invoice `"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Talked to Ruby.Then(after lunch) we left."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Met Ruby.then(later) we left."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Kelly("},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.call(me) later"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Config.Ruby said hi."},
	{"Ruby Kelly called about the invoice.", "Ruby", "System.Ruby called about the invoice."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Notes::Ruby called about the invoice."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby::Kelly called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby::Notes called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "RUBY::NOTES called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "RUBY.CALL(me) later"},
	{"Ruby Kelly called about the invoice.", "Ruby", "See https://linkedin.com/in/Ruby for her profile."},
	{"Ruby Kelly called about the invoice.", "Ruby", "See https://linkedin.com/in/Ruby-Kelly for her profile."},
	{"Ruby Kelly called about the invoice.", "Ruby", "See https://example.com/Ruby/Kelly for her profile."},
	{"Ruby Kelly called about the invoice.", "Ruby", "See https://example.com/?name=Ruby for her profile."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Lunch w/Ruby today."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Files in C:\\Users\\Ruby\\Documents."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Path /home/Ruby/notes.txt"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Path /home/ruby/notes.txt"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Path ~/Ruby is hers."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/Tom/Sam.rb was hers"},
	{"Ruby Kelly called about the invoice.", "Ruby", "a/b/Ruby.rb was hers"},
	{"Ruby Kelly called about the invoice.", "Ruby", "met ruby.she said no"},
	{"Ruby Kelly called about the invoice.", "Ruby", "met ruby.then left"},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby.kelly@ is her"},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby_k said hi"},
	{"Ruby Kelly called about the invoice.", "Ruby", "emailed ruby.k about the invoice"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.a.b= is hers"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Something(else) she said"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Kernel.Ruby called"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/Tom said hi"},
	{"Ruby Kelly called about the invoice.", "Ruby", "RUBY.KELLY("},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby.kelly("},
	{"Ruby Kelly called about the invoice.", "Ruby", "Thanks Ruby.(see above)"},
	{"Ruby Kelly called about the invoice.", "Ruby", "\\Ruby said hi"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby\\ said hi"},
	{"Ruby Kelly called about the invoice.", "Ruby", "the.Ruby.x[ is hers"},
	{"Ruby Kelly called about the invoice.", "Ruby", "See http://Ruby called about the invoice."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/../x said hi"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/x\\ said hi"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Please call Ruby tomorrow`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby called about the invoice`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`thanks Ruby`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`cc Ruby`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "```\nPlease call Ruby tomorrow\n```"},
	{"Ruby Kelly called about the invoice.", "Ruby", "```\nRuby called about the invoice\n```"},
	{"Ruby Kelly called about the invoice.", "Ruby", "```\nthanks Ruby\n```"},
	{"Ruby Kelly called about the invoice.", "Ruby", "```\ncc Ruby\n```"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Kelly called. Use `Ruby` 3.2."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby`'s car broke."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby` and `Kelly` called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ask `Ruby` (Kelly) about it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Kelly, Ruby`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "In the 90` Ruby worked there, in the 2000` she left."},
	{"Ruby Kelly called about the invoice.", "Ruby", "``Ruby'' said hi and ``Tom'' too."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby' said hi and `Tom' too."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby\u2019 said hi."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Escaped \\`Ruby\\` said hi."},
	{"Ruby Kelly called about the invoice.", "Ruby", "a \\`b\\` Ruby said hi \\`c\\`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "I took James` car, then Ruby said hi, then Chris` dog ran."},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby's` gem"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby Kelly`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "`Ruby and Tom`"},
	{"Ruby Kelly called about the invoice.", "Ruby", "```\nx = Ruby\nKelly.save()\n```"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.she said no."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.later we will talk."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Waiting on Ruby.not sure when."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.so tired today."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.and Julia came."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.mom called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.pls call me."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.ok fine."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I texted Ruby.go home now."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.txt me later."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.lock the door."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.md at the clinic called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.c is my sister."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I met Ruby.txt me back."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-test went well for her."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-dev team won the game."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-api called me."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-core is my sister."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby_core called me today."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-setup crew won."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-script for the play"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby:: called me."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby::called me."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/mom_and_dad will drive."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/report.pdf is from her."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Version of events is wrong."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ask about Ruby.Version 2 is due."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Thanks\nRuby.Version helped."},
	{"Ruby Kelly called about the invoice.", "Ruby", "https://facebook.com/Ruby.Kelly"},
	{"Ruby Kelly called about the invoice.", "Ruby", "See the Ruby-lang site."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Load the Ruby_core module."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Read Ruby.Version first."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I can`t reach Ruby today."},
	{"Ruby Kelly called about the invoice.", "Ruby", "She`s with Ruby now."},
	{"Ruby Kelly called about the invoice.", "Ruby", "A lone ` then Ruby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Open `code\n\nRuby called` later."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Mixed ``Ruby` runs."},
	{"Ruby Kelly called about the invoice.", "Ruby", "```\nprint(1)\nRuby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "~~~\nx\n```\nRuby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "    Ruby called me, indented."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Edit my_Ruby_script now."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Use Ruby-3 builds."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Read the Ruby_setup_guide file."},
	{"Julia Kelly called about the invoice.", "Julia", "Notes from Julia-2 review."},
	{"Julia Kelly called about the invoice.", "Julia", "See X-Julia and Julia-x."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/his mom will drive."},
	{"Ruby Kelly called about the invoice.", "Ruby", "@Ruby_Kelly posted again."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Appointments: Ruby 9.15, Julia 10.30."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Dentist: Ruby 9.15."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Pick up Ruby 3.30."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.30 is fine for pickup."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 9.15 and Adam 10.30."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Name: Ruby 9.15"},
	{"Ruby Kelly called about the invoice.", "RUBY", "RUBY 9.15, JULIA 10.30."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 2.0 is my nickname for her."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.2 is out."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I paid Ruby.Then I left."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/Adam/Julia to cover Friday."},
	{"Adam Kelly called about the invoice.", "Adam", "Ruby/Adam/Julia to cover Friday."},
	{"Julia Kelly called about the invoice.", "Julia", "Ruby/Adam/Julia to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/me to cover the shift."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-led audit found nothing."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-approved budget is final."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby language teacher called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby language skills are great."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby interpreter for the court is late."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby runtime at the marathon was 4 hours."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby compiler of the report is out."},
	{"Adam Kelly called.", "Adam", "Adam compiler of the report."},
	{"Julia Kelly called.", "Julia", "Julia interpreter at the hospital."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script was great in the play."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script is done, she wrote it for the play."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version was different."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code was wrong."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code failed the eye test."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I forgot Ruby code."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411."},
	{"Ruby Kelly called.", "Ruby", "* Ruby code\n* Julia code"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby gems are cached."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Check the Ruby version."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The tool is written in Ruby."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code meeting is at noon, the compiler of notes."},
	{"Mary Jenkins called.", "Jenkins", "Jenkins job is at risk after the layoffs."},
	{"Pascal Dupont called.", "Pascal", "The judge dismissed the Pascal case."},
	{"Pascal Dupont called.", "Pascal", "Pascal case was dismissed."},
	{"Jasmine Kelly called.", "Jasmine", "Jasmine tests failed her."},
	{"Hugo Kelly called.", "Hugo", "Hugo site is ready for the visit."},
	{"Apollo Kelly called.", "Apollo", "Apollo server was rude to us."},
	{"Travis Kelly called.", "Travis", "Travis CI is our intern."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Install the ruby gem first."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Upgrade to Ruby 3.3 with rbenv."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.3 is installed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby on Rails is great."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby gems are cached in /var/cache/gems."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code imports the json module."},
	{"Ruby Kelly called about the invoice.", "Ruby", "We deployed the Ruby code to staging."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Run the Ruby script with --verbose."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby interpreter crashed in parse_args."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version 3.3 runs under Docker."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.2 release notes mention YJIT, upgrade soon."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 9.15 release is built nightly."},
	{"Travis Kelly called about the invoice.", "Travis", "Travis CI is red after the npm upgrade."},
	{"Adam Kelly called about the invoice.", "Adam", "Use the Adam optimizer in PyTorch."},
	{"Pascal Kelly called about the invoice.", "Pascal", "Use Pascal case in the userService module."},
	{"Hugo Kelly called about the invoice.", "Hugo", "Hugo site build failed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, she built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ask Ruby (my sister) for the Ruby script she wrote for the play."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events = lies."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Install the app, then call Ruby; Ruby script for the play is ready."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby language teacher -> moved to /home office."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Deploy the troops, Ruby code name is Falcon."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby compiler of the report, see report.pdf."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the building is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for her iPhone is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the gate is 4411, from McDonald's."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for the Spring concert is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about pandas for the zoo trip."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events on 9/15 differs."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for his/her part is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 -- she changed it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, email ruby@gmail.com for help."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events differs, see amazon.com for the receipt."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script is done; she imported the lines from the old play."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 and she got a seat upgrade."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for the play, set in Java, is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Cypress Lane gate is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for The Great Gatsby is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about Kafka for class is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby language tutor lives near the Mars exhibit."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Discord server is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code on YouTube is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events: she had 50/50 odds."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 24/7 for the gate."},
	{"Ruby Kelly called about the invoice.", "Ruby", "\"Ruby code is 4411.\" Then I installed the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411.) Then I installed the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411; install the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411\u3002I installed the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 \U0001f642 installed the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code.Installed the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 10.5 miles away, building closed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.75 GPA, she built a robot."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.30 pickup, install car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 9.15 release from the hospital, install car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby v2 of the essay is done, she built it overnight."},
	{"Ruby Kelly called about the invoice.", "Ruby", "RUBY CODE IS 4411, SHE BUILT THE SHED."},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby code is 4411, she built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby's code is 4411, she built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The letter was written in Ruby handwriting, she built it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The letter was written in Ruby, on the building wall."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby gem of a daughter, she built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby on rails at the station, she built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby syntax in her essay is odd, she built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby runtime at the marathon: 4 hours, she built stamina."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby interpreter at court, debug the schedule."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby lang class, she built a website."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, my_sister knows."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, PIN=0000."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 (see the note())."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, Mom -> school -> home."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 for the C:\\Users folder."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 at ~/home."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby programming language tutor, she built it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for the Python exhibit at the zoo."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about java and donuts for the bakery."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for the Node Street fair."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Apache Junction house is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Windows guy is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events about the Mercury dime."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Atlas Gym is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events: Taylor Swift was there."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events: the Swift concert was loud."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Slack channel is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about a family discord."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the airflow vent is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about the markdown sale."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about the asana poses."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of the fable is shorter."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Django movie night is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about Elixir of Love is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for La Scala is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about the angular face is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about the spark between them."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Puppet show is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for Chef Mario is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about the Jekyll and Hyde play."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the Aurora apartment is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about Prometheus myth."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, she painted the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the gate is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the house is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for the concert is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events differs."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of events on Friday differs."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script about the zoo trip."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411, she changed it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "\"Ruby code is 4411.\" Then I left."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411; call the car seat guy."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.75 GPA, she won a prize."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code for the show is 4411."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby language tutor lives near the exhibit."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The letter was written in Ruby handwriting, she signed it."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby gem of a daughter, she painted the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby script for his part is done."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code is 4411 at home."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version of the story is shorter."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I paid Ruby.Then I built the shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I paid Ruby.Then I installed the car seat."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Call Ruby.ASAP please."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Pay Ruby.OK?"},
	{"Ruby Kelly called about the invoice.", "Ruby", "Called Ruby.Status update tomorrow."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Thanks Ruby.Version 2 of the essay is due."},
	{"Ruby Kelly called about the invoice.", "Ruby", "I saw Ruby.She built a shed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Met Ruby.Data from the lab is in."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Info desk at noon."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/mom will pick up the kids."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/dad to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/the kids at 5."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/2pm pickup."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/his mom will drive."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/sis to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/bf are coming to dinner."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/J.R. to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby/Maria to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "mom/Ruby to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Kids/Ruby at 5."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby / mom to cover Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-supervised trip went well."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-assisted delivery went well."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-requested changes are in."},
	{"Ruby Kelly called about the invoice.", "Ruby", "My ex-Ruby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-mom duo arrived."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-sitting schedule is set."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-hosted dinner is Friday."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby-proofed essay is attached."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby-and-Julia show was fun."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby_Kelly called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "@Ruby_Kelly posted again."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby_k called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby_k called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby_KELLY called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.kelly called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "RUBY.KELLY called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby.k called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Jr called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Kelly.Ruby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "ruby.kelly@gmail.com wrote."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Email ruby.kelly@corp.com today."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Mae called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "r.kelly.ruby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.Smith-Jones called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby__K called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "K_Ruby called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby_K_ called."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby.K. called."},
	{"Mary Jenkins called.", "Jenkins", "Jenkins job is at risk after she deployed overseas."},
	{"Pascal Dupont called.", "Pascal", "Pascal case was dismissed in the building."},
	{"Travis Kelly called.", "Travis", "Travis CI is our intern, he built the shed."},
	{"Apollo Kelly called.", "Apollo", "Apollo server at the diner was rude, he imported cheese."},
	{"Hugo Kelly called.", "Hugo", "Hugo site visit is at the building."},
	{"Jasmine Kelly called.", "Jasmine", "Jasmine tests at school went well, she built a volcano."},
	{"Adam Kelly called.", "Adam", "Adam optimizer of the budget, see budget.xlsx."},
	{"Julia Kelly called.", "Julia", "Julia language tutor at the building is late."},
	{"Julia Kelly called.", "Julia", "Ruby and Julia language tutors are here."},
	{"Julia Kelly called.", "Julia", "Julia language tutor and her Java trip."},
	{"Crystal Kelly called.", "Crystal", "Crystal version of events = lies."},
	{"Ruby Kelly called.", "Ruby", "Ruby 2.5 hours late, building closed."},
	{"Ruby Kelly called.", "Ruby", "Ruby v3 essay done, she built it."},
	{"Ruby Kelly called.", "Ruby", "Ruby 3.30 pickup, install car seat."},
	{"Ruby Kelly called.", "Ruby", "Ruby 12.45 release from the hospital, install car seat."},
	{"Ruby Kelly called.", "Ruby", "Ruby gems, she built a necklace."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Upgrade Ruby to 3.3 with rbenv install."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.3 is installed, run bundle_install next."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Install the Ruby gem with gem install rails."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby interpreter crashed after the upgrade."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby on Rails needs config/database.yml updated."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version in .ruby-version is 3.2.2."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Compile Ruby from source with ./configure."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby script fails with --verbose."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby syntax errors appear after the lint step."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Our Ruby code imports json_parser."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby runtime crashed when we deployed."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby language server needs a rebuild."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The service is written in Ruby and deployed on k8s."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby compiler flags: -O2."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby 3.4 released today, upgrade soon."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Run Ruby code via bin/console."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby gems are cached in vendor/bundle."},
	{"Ruby Kelly called about the invoice.", "Ruby", "The Ruby script calls fetchData() on start."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby version mismatch in Gemfile.lock."},
	{"Ruby Kelly called about the invoice.", "Ruby", "Ruby code coverage dropped after the refactor."},
	{"Julia Kelly called.", "Julia", "Julia 1.10 is installed via juliaup."},
	{"Julia Kelly called.", "Julia", "The Julia compiler is slow on first import."},
	{"Julia Kelly called.", "Julia", "Install Julia language support in VS Code."},
	{"Julia Kelly called.", "Julia", "Julia code runs faster after the precompile_all step."},
	{"Julia Kelly called.", "Julia", "Julia version in Project.toml is 1.9."},
	{"Mary Jenkins called.", "Jenkins", "Jenkins pipeline failed after the deploy."},
	{"Mary Jenkins called.", "Jenkins", "The Jenkins job runs build.sh nightly."},
	{"Mary Jenkins called.", "Jenkins", "Jenkins version 2.4 needs an upgrade."},
	{"Travis Kelly called.", "Travis", "Travis CI is red after the npm upgrade."},
	{"Travis Kelly called.", "Travis", "Travis CI runs make test on each push."},
	{"Apollo Kelly called.", "Apollo", "Apollo client cache is imported from lib/cache.ts"},
	{"Apollo Kelly called.", "Apollo", "Apollo server version 4.1 is deployed."},
	{"Hugo Kelly called.", "Hugo", "Hugo site build failed."},
	{"Hugo Kelly called.", "Hugo", "Hugo version 0.120 is installed."},
	{"Jasmine Kelly called.", "Jasmine", "Write Jasmine tests and run them with --watch."},
	{"Jasmine Kelly called.", "Jasmine", "Jasmine test suite fails on import."},
	{"Adam Kelly called.", "Adam", "Use the Adam optimizer in PyTorch."},
	{"Adam Kelly called.", "Adam", "The Adam optimizer config sets lr=0.001."},
	{"Mary Docker called.", "Docker", "Docker compose up fails after the upgrade."},
	{"Mary Docker called.", "Docker", "Docker version 24.0 is installed."},
	{"Pascal Dupont called.", "Pascal", "Rename my_var to Pascal case for the lint rule."},
	{"Pascal Dupont called.", "Pascal", "Pascal compiler output is in build/out."},
	{"Mary Kafka called.", "Kafka", "Kafka topic config uses retention_ms."},
	{"Mary Kafka called.", "Kafka", "Kafka version 3.6 is deployed."},
}

var homonymCodeKeeps = []struct{ seed, in string }{
	{"Ruby Kelly called about the invoice.", "Use `Ruby.run(1)` here."},
	{"Ruby Kelly called about the invoice.", "Use `Ruby::Version` here."},
	{"Ruby Kelly called about the invoice.", "Call `Config.Ruby` now."},
	{"Ruby Kelly called about the invoice.", "Open `lib/Ruby/core.rb` please."},
	{"Ruby Kelly called about the invoice.", "See `https://example.com/Ruby/docs` today."},
	{"Ruby Kelly called about the invoice.", "`Ruby` 3.2 is out."},
	{"Ruby Kelly called about the invoice.", "`Ruby` called me yesterday."},
	{"Crystal Kelly called about the invoice.", "`System.Crystal` fails."},
	{"Ruby Kelly called about the invoice.", "`Ruby` said hi."},
	{"Ruby Kelly called about the invoice.", "`Ruby build`"},
	{"Ruby Kelly called about the invoice.", "``Ruby``"},
	{"Ruby Kelly called about the invoice.", "`` `Ruby` ``"},
	{"Ruby Kelly called about the invoice.", "Load ruby_core now."},
	{"Ruby Kelly called about the invoice.", "~~~\nRuby.run(1)\nx = Ruby\n~~~"},
	{"Ruby Kelly called about the invoice.", "````\nx\n```\nx = Ruby\n````"},
	{"Ruby Kelly called about the invoice.", "Use ``Ruby`` here."},
	{"Ruby Kelly called about the invoice.", "Run `Ruby --version` first."},
	{"Ruby Kelly called about the invoice.", "```\nRuby.run(1)\n```"},
	{"Ruby Kelly called about the invoice.", "Edit my_ruby_script.rb now."},
	{"Ruby Kelly called about the invoice.", "See ruby-lang for the docs."},
	{"Ruby Kelly called about the invoice.", "Open ruby.yml please."},
}

func rawNameLeft(out, name string) bool {
	for _, f := range strings.FieldsFunc(out, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if strings.EqualFold(f, name) && f != strings.ToLower(f) {
			return true
		}
	}
	return false
}

func TestKnownHomonymMasksInAllProse(t *testing.T) {
	ctx := context.Background()
	for _, c := range homonymProseMasks {
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			if rawNameLeft(out, c.name) {
				t.Errorf("%s: %q left raw as %q", mode, c.in, out)
			}
			if back != c.in && !strings.Contains(out, "[redacted") {
				t.Errorf("%s: %q restored as %q", mode, c.in, back)
			}
		})
		s := newEngine(t, nil).NewSession("p")
		in := c.seed + " " + c.in
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if rawNameLeft(out, c.name) {
			t.Errorf("anchored: %q left raw as %q", in, out)
		}
	}
}

func TestNamesAroundLongCodeBlocks(t *testing.T) {
	ctx := context.Background()
	for _, n := range []int{300, 1000, 10000} {
		body := strings.Repeat("x1 = compute_value(1) + offset\n", n/31+1)
		for _, in := range []string{
			"```python\n" + body + "```\nRuby called me about the refund. Ruby is upset.",
			"Here is the log:\n```\n" + body + "```\nPlease tell Ruby the fix is in. Ruby wants an update.",
			"```\n" + body + "Ruby called. Ruby is upset.",
		} {
			strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
				if rawNameLeft(out, "Ruby") {
					t.Errorf("%s %d: a name after the block stayed raw", mode, n)
				}
				if back != in {
					t.Errorf("%s %d: not restored", mode, n)
				}
			})
		}
		inside := "```\n" + body + "x = Ruby\n```\nok"
		strictMaskModes(t, "Ruby Kelly called about the invoice.", inside, func(mode, out, back string) {
			if out != inside {
				t.Errorf("%s %d: a name inside a closed block changed", mode, n)
			}
		})
		out, err := newEngine(t, nil).NewSession("p").Mask(ctx, "Ruby Kelly called about the invoice. "+"```python\n"+body+"```\nRuby called me.")
		if err != nil {
			t.Fatal(err)
		}
		if rawNameLeft(out, "Ruby") {
			t.Errorf("anchored %d: a name after the block stayed raw", n)
		}
	}
}

func TestKnownHomonymKeepsOnlyInCode(t *testing.T) {
	ctx := context.Background()
	for _, c := range homonymCodeKeeps {
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			if out != c.in {
				t.Errorf("%s: %q became %q", mode, c.in, out)
			}
		})
		as := newEngine(t, nil).NewSession("p")
		if _, err := as.Mask(ctx, c.seed); err != nil {
			t.Fatal(err)
		}
		if out, _ := as.Mask(ctx, c.in); out != c.in {
			t.Errorf("anchored session: %q became %q", c.in, out)
		}
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, c.seed+"\n\n"+c.in); out == c.seed+"\n\n"+c.in {
			t.Errorf("same text: a masked mention did not void the keep in %q", out)
		}
	}
}

func TestLowercaseHomonymFollowsCapitalizedOnlyRule(t *testing.T) {
	ctx := context.Background()
	for _, in := range []string{"Install the ruby gem first.", "ruby code is 4411, she built the shed."} {
		strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
			if !hasWholeWord(out, "ruby") {
				t.Errorf("%s: lowercase %q became %q", mode, in, out)
			}
		})
	}
	as := newEngine(t, nil).NewSession("p")
	as.Mask(ctx, "Ruby Kelly called about the invoice.")
	if out, _ := as.Mask(ctx, "Install the ruby gem first."); out != "Install the ruby gem first." {
		t.Errorf("anchored session lowercase became %q", out)
	}
	out, err := newEngine(t, nil).NewSession("p").Mask(ctx, "Ruby Kelly called about the invoice. Install the ruby gem first.")
	if err != nil {
		t.Fatal(err)
	}
	if hasWholeWord(out, "ruby") {
		t.Errorf("anchored lowercase in the same paragraph stayed raw: %q", out)
	}
}

func TestKnownNameHandleAndWidthFormsMask(t *testing.T) {
	cases := []struct{ seed, in string }{
		{"Ruby Kelly called about the invoice.", "Ping ruby.kelly about it."},
		{"Ruby Kelly called about the invoice.", "ruby_kelly pushed a fix."},
		{"Anna Kelly called about the invoice.", "Ping anna.kelly about it."},
		{"Ruby Kelly called about the invoice.", "Ruby.K wrote back."},
		{"Ruby Kelly called about the invoice.", "Ruby_K wrote back."},
		{"Ruby Kelly called about the invoice.", "K_Ruby wrote back."},
		{"Ruby Kelly called about the invoice.", "\uff32uby wants a refund."},
		{"Ruby Kelly called about the invoice.", "I paid \uff32uby yesterday."},
	}
	ctx := context.Background()
	for _, c := range cases {
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			low := strings.ToLower(out)
			for _, w := range []string{"ruby", "anna", "kelly", "\uff52uby"} {
				if hasWholeWord(low, w) {
					t.Errorf("%s: %q left %s raw in %q", mode, c.in, w, out)
				}
			}
		})
		out, err := newEngine(t, nil).NewSession("p").Mask(ctx, c.seed+" "+c.in)
		if err != nil {
			t.Fatal(err)
		}
		if low := strings.ToLower(out); hasWholeWord(low, "ruby") || hasWholeWord(low, "anna") || hasWholeWord(low, "\uff52uby") {
			t.Errorf("anchored %q left raw in %q", c.in, out)
		}
	}
	for _, in := range []string{"Open ruby.rake and ruby.yml now.", "Use gem.kelly now."} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, "Ruby Kelly called about the invoice."); err != nil {
			t.Fatal(err)
		}
		out, _ := e.NewSession("later").Mask(ctx, in)
		if strings.Contains(in, "ruby.rake") && out != in {
			t.Errorf("file names changed: %q became %q", in, out)
		}
		if strings.Contains(in, "gem.") && !strings.Contains(out, "gem.") {
			t.Errorf("%q lost its non-name part: %q", in, out)
		}
	}
}

func TestVaultPairsMaskGluedAndLowercaseForms(t *testing.T) {
	cases := []struct {
		seed, in string
		raw      []string
	}{
		{"Anna Kelly called about the invoice.", "annakelly called.", []string{"annakelly"}},
		{"Anna Kelly called about the invoice.", "AnnaKelly called.", []string{"AnnaKelly"}},
		{"Anna Kelly called about the invoice.", "kellyanna called.", []string{"kellyanna"}},
		{"Anna Kelly called about the invoice.", "anna kelly called me.", []string{"anna", "kelly"}},
		{"Anna Kelly called about the invoice.", "talked to anna kelly today.", []string{"anna", "kelly"}},
		{"Ruby Kelly called about the invoice.", "rubykelly called.", []string{"rubykelly"}},
		{"Ruby Kelly called about the invoice.", "RubyKelly called.", []string{"RubyKelly"}},
		{"Ruby Kelly called about the invoice.", "kellyruby called.", []string{"kellyruby"}},
		{"Ruby Kelly called about the invoice.", "ruby_kelly called.", []string{"ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "ruby-kelly called.", []string{"ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "ruby.kelly called.", []string{"ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "@ruby_kelly posted again.", []string{"ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "ruby kelly called me.", []string{"ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "talked to ruby kelly today.", []string{"ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "ruby Kelly called me.", []string{"ruby", "Kelly"}},
		{"Ruby Kelly called about the invoice.", "Ruby kelly called me.", []string{"Ruby", "kelly"}},
		{"Ruby Kelly called about the invoice.", "RUBY KELLY called me.", []string{"RUBY", "KELLY"}},
	}
	ctx := context.Background()
	for _, c := range cases {
		check := func(mode, out string) {
			for _, w := range c.raw {
				if hasWholeWord(out, w) {
					t.Errorf("%s: %q left %s raw in %q", mode, c.in, w, out)
				}
			}
		}
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			check(mode, out)
			if back != c.in {
				t.Errorf("%s: %q restored as %q", mode, c.in, back)
			}
		})
		s := newEngine(t, nil).NewSession("p")
		in := c.seed + " " + c.in
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		check("anchored", out)
		if back := s.Rehydrate(out); back != in {
			t.Errorf("anchored: %q restored as %q", in, back)
		}
	}
	for _, in := range []string{"ruby called me.", "my sister ruby called.", "thanks ruby!", "rubykellyson called.", "The rubykellyson file is here."} {
		strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
			if mode != "same" && out != in {
				t.Errorf("%s: %q became %q", mode, in, out)
			}
		})
	}
}

func TestCapitalizedTokensMaskOutsideCode(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct{ seed, token, name string }{
		{"Ruby Kelly called about the invoice.", "Config.Ruby", "Ruby"},
		{"Ruby Kelly called about the invoice.", "Module.Ruby", "Ruby"},
		{"Crystal Kelly called about the invoice.", "System.Crystal", "Crystal"},
		{"Ruby Kelly called about the invoice.", "Ruby::Version", "Ruby"},
		{"Ruby Kelly called about the invoice.", "Ruby.run(1)", "Ruby"},
		{"Ruby Kelly called about the invoice.", "lib/Ruby/core.rb", "Ruby"},
		{"Ruby Kelly called about the invoice.", "/usr/lib/Ruby/core", "Ruby"},
		{"Ruby Kelly called about the invoice.", "./Ruby/setup", "Ruby"},
		{"Ruby Kelly called about the invoice.", "https://linkedin.com/in/Ruby", "Ruby"},
	} {
		for _, in := range []string{"call " + c.token + " now"} {
			strictMaskModes(t, c.seed, in, func(mode, out, back string) {
				if rawNameLeft(out, c.name) {
					t.Errorf("%s: %q left raw as %q", mode, in, out)
				}
				if back != in {
					t.Errorf("%s: %q restored as %q", mode, in, back)
				}
			})
			s := newEngine(t, nil).NewSession("p")
			full := c.seed + "\n\n" + in
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			if rawNameLeft(out, c.name) {
				t.Errorf("anchored: %q left raw as %q", in, out)
			}
			if back := s.Rehydrate(out); back != full {
				t.Errorf("anchored: %q restored as %q", full, back)
			}
		}
		code := "call `" + c.token + "` now"
		strictMaskModes(t, c.seed, code, func(mode, out, back string) {
			if out != code {
				t.Errorf("%s: %q became %q", mode, code, out)
			}
		})
	}
}

func TestMustKeepTokensInProseAllModes(t *testing.T) {
	ctx := context.Background()
	cases := []struct{ seed, token string }{
		{"Ruby Kelly called about the invoice.", "ruby-lang"},
		{"Ruby Kelly called about the invoice.", "ruby.yml"},
		{"Ruby Kelly called about the invoice.", "my_ruby_script.rb"},
		{"Ruby Kelly called about the invoice.", "`RUBY_VERSION`"},
	}
	for _, c := range cases {
		in := "call " + c.token + " now"
		strictMaskModes(t, c.seed, in, func(mode, out, back string) {
			if out != in {
				t.Errorf("%s: %q became %q", mode, in, out)
			}
		})
		as := newEngine(t, nil).NewSession("p")
		if _, err := as.Mask(ctx, c.seed); err != nil {
			t.Fatal(err)
		}
		if out, _ := as.Mask(ctx, in); out != in {
			t.Errorf("anchored session: %q became %q", in, out)
		}
	}
}

func TestIndentedFencesAndCapsSnakeMask(t *testing.T) {
	ctx := context.Background()
	cases := []struct{ seed, in, raw string }{
		{"Ruby Kelly called about the invoice.", "Run this:\n    ```\n    Ruby build\n    ```\nok", "Ruby"},
		{"Ruby Kelly called about the invoice.", "Run this:\n\t```\n\tRuby build\n\t```\nok", "Ruby"},
		{"Ruby Kelly called about the invoice.", "```\nx = 1\n    ```\nRuby build", "Ruby"},
		{"Ruby Kelly called about the invoice.", "Set RUBY_VERSION before the trip.", "RUBY"},
		{"Anna Kelly called about the invoice.", "Use the kelly green paint.", "kelly"},
	}
	for _, c := range cases {
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			if hasWholeWord(out, c.raw) {
				t.Errorf("%s: %q left %s raw in %q", mode, c.in, c.raw, out)
			}
			if back != c.in {
				t.Errorf("%s: %q restored as %q", mode, c.in, back)
			}
		})
		s := newEngine(t, nil).NewSession("p")
		in := c.seed + "\n" + c.in
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if hasWholeWord(out[strings.IndexByte(out, '\n'):], c.raw) {
			t.Errorf("anchored: %q left %s raw in %q", c.in, c.raw, out)
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("anchored: %q restored as %q", in, back)
		}
	}
}

func TestGluedPairMasksStablyAndRoundTrips(t *testing.T) {
	ctx := context.Background()
	cases := []struct{ seed, in, raw string }{
		{"Anna Kelly called about the invoice.", "AnnaKelly called.", "AnnaKelly"},
		{"Anna Kelly called about the invoice.", "KellyAnna called.", "KellyAnna"},
		{"Ruby Kelly called about the invoice.", "RubyKelly called.", "RubyKelly"},
		{"Anna Kelly called about the invoice.", "ANNAKELLY called.", "ANNAKELLY"},
		{"Anna Kelly called about the invoice.", "annakelly called.", "annakelly"},
	}
	stream := func(s *Session, out string) string {
		r := s.NewStreamRehydrator(false)
		var b strings.Builder
		for i := 0; i < len(out); i++ {
			b.WriteString(r.Push(out[i : i+1]))
		}
		b.WriteString(r.Flush())
		return b.String()
	}
	camel := func(w string) bool {
		inner := 0
		for i, r := range w {
			if i > 0 && unicode.IsUpper(r) {
				inner++
			}
		}
		return inner == 1 && unicode.IsUpper([]rune(w)[0])
	}
	for _, c := range cases {
		for _, mode := range []string{"same", "new", "reopen", "anchored"} {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			seed := e.NewSession("one")
			if mode != "anchored" {
				if _, err := seed.Mask(ctx, c.seed); err != nil {
					t.Fatal(err)
				}
			}
			in := c.in
			s := seed
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v = strictFileEngine(t, path)
				s = e.NewSession("two")
			case "anchored":
				in = c.seed + " " + c.in
			}
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if hasWholeWord(out, c.raw) {
				t.Errorf("%s: %q left raw in %q", mode, in, out)
			}
			fields := strings.Fields(out)
			masked := fields[len(fields)-2]
			if camel(c.raw) && (camel(masked) || masked != textutil.MatchCase(c.raw, strings.ToLower(masked))) {
				t.Errorf("%s: %q glued surrogate is not in the byte-stable capitalized form: %q", mode, in, out)
			}
			if back := s.Rehydrate(out); back != in {
				t.Errorf("%s batch: %q restored as %q", mode, in, back)
			}
			if back := stream(s, out); back != in {
				t.Errorf("%s stream: %q restored as %q", mode, in, back)
			}
			v.Close()
		}
	}
}

func TestPairInOtherParagraphKeepsSpan(t *testing.T) {
	in := "Kelly called.\n\nUse `Ruby` 3.2."
	strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
		if hasWholeWord(out, "Kelly") || !strings.Contains(out, "`Ruby` 3.2") {
			t.Errorf("%s: %q became %q", mode, in, out)
		}
		if back != in {
			t.Errorf("%s: %q restored as %q", mode, in, back)
		}
	})
}

func gluedCase(w, pattern string) string {
	switch pattern {
	case "lower":
		return strings.ToLower(w)
	case "upper":
		return strings.ToUpper(w)
	}
	return titleWord(w)
}

func TestGluedPairFormsMaskAndRestoreEverywhere(t *testing.T) {
	ctx := context.Background()
	seeds := []struct{ seed, given, family string }{
		{"Ruby Kelly and Anna Moss called about the invoice.", "Ruby", "Kelly"},
		{"Ruby Kelly and Anna Moss called about the invoice.", "Anna", "Moss"},
		{"Zoë Kelly called about the invoice.", "Zoë", "Kelly"},
		{"José Ramírez called about the invoice.", "José", "Ramírez"},
	}
	patterns := [][2]string{{"lower", "lower"}, {"title", "title"}, {"upper", "upper"}, {"lower", "title"}, {"upper", "lower"}}
	affixes := [][2]string{{"", ""}, {"", "99"}, {"2", ""}, {"", "_"}, {"@", ""}, {"", "s"}}
	stream := func(s *Session, out string) string {
		r := s.NewStreamRehydrator(false)
		var b strings.Builder
		for i := 0; i < len(out); i++ {
			b.WriteString(r.Push(out[i : i+1]))
		}
		b.WriteString(r.Flush())
		return b.String()
	}
	for _, sd := range seeds {
		g, f := textutil.Fold(sd.given), textutil.Fold(sd.family)
		for oi, order := range [][2]string{{sd.given, sd.family}, {sd.family, sd.given}} {
			for pi, p := range patterns {
				for ai, af := range affixes {
					if !raceKeep(t, oi*100+pi*10+ai, 3) {
						continue
					}
					word := af[0] + gluedCase(order[0], p[0]) + gluedCase(order[1], p[1]) + af[1]
					in := "ping " + word + " today"
					leak := func(mode, out string) {
						tok := strings.TrimSuffix(strings.TrimPrefix(out, "ping "), " today")
						tok = strings.TrimFunc(textutil.Fold(tok), func(r rune) bool { return !unicode.IsLetter(r) })
						tok = strings.TrimSuffix(tok, "s")
						if strings.HasPrefix(tok, g) || strings.HasPrefix(tok, f) || strings.HasSuffix(tok, g) || strings.HasSuffix(tok, f) {
							t.Errorf("%s: %q left a real part in %q", mode, in, out)
						}
						if len(tok) < 8 {
							t.Errorf("%s: %q got a fragment surrogate in %q", mode, in, out)
						}
					}
					for _, mode := range []string{"same", "new", "reopen", "anchored"} {
						path := filepath.Join(t.TempDir(), "v.db")
						e, v := modeEngine(t, mode, path)
						seed := e.NewSession("one")
						if mode != "anchored" {
							if _, err := seed.Mask(ctx, sd.seed); err != nil {
								t.Fatal(err)
							}
						}
						s, full := seed, in
						switch mode {
						case "new":
							s = e.NewSession("two")
						case "reopen":
							v.Close()
							e, v = strictFileEngine(t, path)
							s = e.NewSession("two")
						case "anchored":
							full = sd.seed + "\n\n" + in
						}
						out, err := s.Mask(ctx, full)
						if err != nil {
							t.Fatal(err)
						}
						leak(mode, out[strings.LastIndex(out, "ping "):])
						if back := s.Rehydrate(out); back != full {
							t.Errorf("%s batch: %q restored as %q", mode, full, back)
						}
						if back := stream(s, out); back != full {
							t.Errorf("%s stream: %q restored as %q", mode, full, back)
						}
						v.Close()
					}
				}
			}
		}
	}
}

var pairedParagraphRows = []struct {
	seed, in string
	rawOK    bool
}{
	{"Ruby Kelly called about the invoice.", "ruby kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby  kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby\tkelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby\nkelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby.kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby_kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby-kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby/kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby,kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby kelly's car", false},
	{"Ruby Kelly called about the invoice.", "RUBY KELLY called", false},
	{"Ruby Kelly called about the invoice.", "ruby KELLY called", false},
	{"Ruby Kelly called about the invoice.", "ruby Kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby k. called", true},
	{"Ruby Kelly called about the invoice.", "ruby kellys called", false},
	{"Ruby Kelly called about the invoice.", "ruby  Kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby   kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby\u00a0kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby\u2009kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby\u3000kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby - kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby, kelly called", false},
	{"Ruby Kelly called about the invoice.", "hi ruby kelly", false},
	{"Ruby Kelly called about the invoice.", "kelly ruby called", false},
	{"Ruby Kelly called about the invoice.", "kelly, ruby called", false},
	{"Ruby Kelly called about the invoice.", "ruby (kelly) called", false},
	{"Ruby Kelly called about the invoice.", "ruby\u00b7kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby+kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby&kelly called", false},
	{"Ruby Kelly called about the invoice.", "ruby@kelly called", false},
	{"Crystal Moss called about the invoice.", "crystal moss called", false},
	{"Crystal Moss called about the invoice.", "crystal  moss called", false},
	{"Crystal Moss called about the invoice.", "crystal.moss called", false},
	{"Crystal Moss called about the invoice.", "crystal\tmoss called", false},
	{"Julia Stone called about the invoice.", "julia stone called", false},
	{"Julia Stone called about the invoice.", "julia  stone called", false},
	{"Julia Stone called about the invoice.", "julia_stone called", false},
	{"Julia Stone called about the invoice.", "Julia.Stone( called", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "kellyruby said feny is fine.", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "kellyruby said paly is fine.", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "mossanna said amay and nady are fine.", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "feny", true},
	{"Ruby Kelly and Anna Moss called about the invoice.", "kellyruby mossanna", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "kellyruby and kellyruby again", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "Kellyruby said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "KELLYRUBY said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "kellyruby99 said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "ruby.kelly99 said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "ruby_kelly_99 said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "rubykelly1990@gmail.com wrote", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "x_rubykelly said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "rubykelly_x said hi", false},
	{"Ruby Kelly and Anna Moss called about the invoice.", "user rubykellyx said hi", true},
	{"Ruby Kelly called about the invoice.", "Kelly called. install ruby today", false},
}

func realNameRunLeft(out string, names []string) bool {
	set := map[string]bool{}
	for _, a := range names {
		set[a] = true
		for _, b := range names {
			if a != b {
				set[a+b] = true
			}
		}
	}
	for _, f := range strings.FieldsFunc(textutil.Fold(out), func(r rune) bool { return r < 'a' || r > 'z' }) {
		if set[f] || set[strings.TrimSuffix(f, "s")] {
			return true
		}
	}
	return false
}

func TestPairedParagraphMasksEveryJoinerAndRestores(t *testing.T) {
	ctx := context.Background()
	for _, c := range pairedParagraphRows {
		names := strings.Fields(textutil.Fold(strings.NewReplacer(" called about the invoice.", "", " and ", " ").Replace(c.seed)))
		check := func(mode, out string) {
			if !c.rawOK && realNameRunLeft(out, names) {
				t.Errorf("%s: %q left a real name in %q", mode, c.in, out)
			}
		}
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			check(mode, out)
			if back != c.in && !strings.Contains(out, "[redacted") {
				t.Errorf("%s: %q restored as %q", mode, c.in, back)
			}
		})
		s := newEngine(t, nil).NewSession("p")
		full := c.seed + " " + c.in
		out, err := s.Mask(ctx, full)
		if err != nil {
			t.Fatal(err)
		}
		check("anchored", out)
		if back := s.Rehydrate(out); back != full && !strings.Contains(out, "[redacted") {
			t.Errorf("anchored: %q restored as %q", full, back)
		}
	}
	for _, in := range []string{"install ruby today", "Kelly called.\n\ninstall ruby today"} {
		strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
			if mode != "same" && !hasWholeWord(out, "ruby") {
				t.Errorf("%s: lone lowercase ruby in %q became %q", mode, in, out)
			}
			if back != in {
				t.Errorf("%s: %q restored as %q", mode, in, back)
			}
		})
	}
}

func foldedLetters(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		return -1
	}, textutil.Fold(s))
}

func spellingStream(s *Session, out string) string {
	r := s.NewStreamRehydrator(false)
	var b strings.Builder
	for i := 0; i < len(out); i++ {
		b.WriteString(r.Push(out[i : i+1]))
	}
	b.WriteString(r.Flush())
	return b.String()
}

func TestSpellingVariantsMaskAndRestoreExactly(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		seed, in string
		rawOK    []string
	}{
		{"Ruby Strauß called about the invoice.", "ruby strauss called.", nil},
		{"Ruby Strauss called about the invoice.", "ruby strauß called.", nil},
		{"Ruby Strauß called about the invoice.", "Ruby Strauß met ruby strauss today.", nil},
		{"Zoë Kelly called about the invoice.", "ping zoe kelly today", nil},
		{"Zoë Kelly called about the invoice.", "ping Zoe Kelly today", nil},
		{"Zoë Kelly called about the invoice.", "ping Zoe today", nil},
		{"Zoë Kelly called about the invoice.", "Zoë Kelly and Zoe Kelly met.", nil},
		{"Zoe Kelly called about the invoice.", "ping ZOË today", nil},
		{"Zoe Kelly called about the invoice.", "ZOË called.", nil},
		{"Zoe Kelly called about the invoice.", "Zoë called.", nil},
		{"Jose Ortega called about the invoice.", "ping JOSÉ today", nil},
		{"José Ramírez called about the invoice.", "ping Jose Ramirez today", nil},
		{"Søren Kelly called about the invoice.", "ping Soren Kelly today", nil},
		{"Zoë Kelly called about the invoice.", "ping Zoë Kelly today", nil},
		{"Zoë Kelly called about the invoice.", "ping Zoë Kelly today", nil},
		{"Ruby Kelly called about the invoice.", "ping Ｒuby Kelly today", nil},
		{"Ruby Kelly called about the invoice.", "ping rUBYkELLY today", nil},
		{"Ruby Kelly called about the invoice.", "ping rUbYkElLy today", nil},
		{"Ruby Kelly called about the invoice.", "ping rubykellY today", nil},
		{"Ruby Kelly called about the invoice.", "ping rUBY kELLY today", nil},
		{"Ruby O'Brien called about the invoice.", "rubyobrien said hi", nil},
		{"Ruby O'Brien called about the invoice.", "ruby obrien called me.", nil},
		{"Ruby Smith-Jones called about the invoice.", "rubysmithjones said hi", nil},
		{"Ruby Smith-Jones called about the invoice.", "ruby smithjones called me.", nil},
		{"Will Smith called about the invoice.", "will smith called me.", nil},
		{"Will Smith called about the invoice.", "will  smith called me.", nil},
		{"Will Smith called about the invoice.", "willsmith said hi", nil},
		{"Will Smith called about the invoice.", "will.smith99 said hi", nil},
		{"May Kelly called about the invoice.", "may kelly called me.", nil},
		{"Will Smith called about the invoice.", "i saw will and smith today.", []string{"will"}},
		{"Ruby Kelly called about the invoice.", "`Ruby` and `Ruby`'s car broke.", nil},
		{"Ruby Kelly called about the invoice.", "Ruby called.\n\n```\nx = Ruby\n```", nil},
		{"Ruby Kelly called about the invoice.", "``Ruby at noon'' and ``lunch after'' she wrote.", nil},
		{"Ruby Kelly called about the invoice.", "`Ruby at noon' and `lunch after' she wrote.", nil},
		{"Ruby Kelly called about the invoice.", "`Ruby` 's car broke.", nil},
	}
	for _, c := range cases {
		var names []string
		for _, w := range strings.Fields(strings.TrimSuffix(c.seed, " called about the invoice.")) {
			names = append(names, foldedLetters(w))
		}
		allowed := map[string]bool{}
		for _, w := range c.rawOK {
			allowed[w] = true
		}
		leak := func(mode, out string) {
			for _, tok := range strings.FieldsFunc(textutil.Fold(out), func(r rune) bool { return r < 'a' || r > 'z' }) {
				for _, n := range names {
					if !allowed[tok] && (tok == n || strings.HasPrefix(tok, n) && len(tok) > len(n) && len(n) > 3 && (strings.Contains(tok[len(n):], names[len(names)-1]) || strings.Contains(tok[len(n):], names[0]))) {
						t.Errorf("%s: %q left %q raw in %q", mode, c.in, tok, out)
					}
				}
			}
		}
		for _, mode := range []string{"same", "new", "reopen", "anchored"} {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := strictFileEngine(t, path)
			seed := e.NewSession("one")
			if mode != "anchored" {
				if _, err := seed.Mask(ctx, c.seed); err != nil {
					t.Fatal(err)
				}
			}
			s, full := seed, c.in
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v = strictFileEngine(t, path)
				s = e.NewSession("two")
			case "anchored":
				full = c.seed + " " + c.in
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			check := out
			if mode == "anchored" {
				check = out[strings.Index(out, "invoice.")+len("invoice."):]
			}
			leak(mode, check)
			if back := s.Rehydrate(out); back != full {
				t.Errorf("%s batch: %q restored as %q", mode, full, back)
			}
			if back := spellingStream(s, out); back != full {
				t.Errorf("%s stream: %q restored as %q", mode, full, back)
			}
			v.Close()
			e2, v2 := strictFileEngine(t, path)
			if back := e2.NewSession("cli").Rehydrate(out); back != full {
				t.Errorf("%s fresh engine: %q restored as %q", mode, full, back)
			}
			v2.Close()
		}
	}
	for _, in := range []string{"```\nx = Ruby\n```\n\nuse `Ruby` here"} {
		strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
			if out != in {
				t.Errorf("%s: control %q became %q", mode, in, out)
			}
		})
	}
}

func TestFirstSightParticlesAndMarks(t *testing.T) {
	for _, c := range []struct{ in, raw string }{
		{"Ruby van der Berg called about the invoice.", "Berg"},
		{"Anna van der Berg called about the invoice.", "Berg"},
		{"Ruby de la Cruz called about the invoice.", "Cruz"},
		{"Zoë Kelly called about the invoice.", "Kelly"},
		{"Then Zoë Kelly called.", "Zoe"},
	} {
		e := newEngine(t, nil)
		s := e.NewSession("p")
		out, err := s.Mask(context.Background(), c.in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, c.raw) || strings.ContainsRune(out, '̈') {
			t.Errorf("%q -> %q", c.in, out)
		}
		if back := s.Rehydrate(out); back != c.in {
			t.Errorf("%q restored as %q", c.in, back)
		}
	}
	for in, raw := range map[string]bool{"Path /home/ruby/notes.txt": true, "Kelly called. Path /home/ruby/notes.txt": false} {
		strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
			if hasWholeWord(out, "ruby") != raw {
				t.Errorf("%s: %q became %q", mode, in, out)
			}
			if back != in {
				t.Errorf("%s: %q restored as %q", mode, in, back)
			}
		})
	}
}

func TestRulingsRuleFiveBeatsIdentifierAndFenceKeeps(t *testing.T) {
	for _, in := range []string{"Kelly called. edit ruby.yml now", "Kelly called. see ruby-lang docs", "Kelly called. run my_ruby_script.rb"} {
		strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
			if strings.Contains(strings.ToLower(out), "ruby") || strings.Contains(out, "Kelly") {
				t.Errorf("%s: %q became %q", mode, in, out)
			}
			if back != in {
				t.Errorf("%s: %q restored as %q", mode, in, back)
			}
		})
	}
	in := "Ask Kelly.\n```\nRuby build\n```"
	strictMaskModes(t, "Ruby Kelly called about the invoice.", in, func(mode, out, back string) {
		if !strings.Contains(out, "```\nRuby build\n```") {
			t.Errorf("%s: fence keep lost in %q", mode, out)
		}
		if back != in {
			t.Errorf("%s: %q restored as %q", mode, in, back)
		}
	})
}

func TestTurkishNamesMaskAtFirstSight(t *testing.T) {
	for _, c := range []struct{ in, a, b string }{
		{"Işıl Kaya called about the invoice.", "Işıl", "Kaya"},
		{"İlker Demir called.", "İlker", "Demir"},
		{"IŞIL KAYA called about the invoice.", "IŞIL", "KAYA"},
		{"İLKER DEMİR called.", "İLKER", "DEMİR"},
		{"Please call Işık Yılmaz today.", "Işık", "Yılmaz"},
		{"Merhaba, Mert Çelik and Ebru Şahin joined the call.", "Mert", "Şahin"},
	} {
		s := newEngine(t, nil).NewSession("p")
		out, err := s.Mask(context.Background(), c.in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, c.a) || strings.Contains(out, c.b) {
			t.Errorf("%q -> %q", c.in, out)
		}
		if back := s.Rehydrate(out); back != c.in {
			t.Errorf("%q restored as %q", c.in, back)
		}
	}
}

func caseForm(w, form string) string {
	switch form {
	case "lower":
		return strings.ToLower(w)
	case "upper":
		return strings.ToUpper(w)
	case "irregular":
		rs := []rune(strings.ToLower(w))
		for i := range rs {
			if i%2 == 1 {
				rs[i] = unicode.ToUpper(rs[i])
			}
		}
		return string(rs)
	}
	return titleWord(w)
}

func TestPairMasksBothHalvesAcrossSpellingsAndCases(t *testing.T) {
	ctx := context.Background()
	pairs := []struct {
		seedGiven, seedFamily string
		givens, families      []string
	}{
		{"Zoe", "Kelly", []string{"Zoe", "Zoë"}, []string{"Kelly"}},
		{"Zoë", "Kelly", []string{"Zoe", "Zoë"}, []string{"Kelly"}},
		{"José", "Ramírez", []string{"Jose", "José"}, []string{"Ramirez", "Ramírez"}},
		{"Anna", "Strauß", []string{"Anna"}, []string{"Strauss", "Strauß"}},
	}
	forms := []string{"lower", "title", "upper", "irregular"}
	for _, p := range pairs {
		seed := p.seedGiven + " " + p.seedFamily + " called about the invoice."
		gk, fk := foldedLetters(p.seedGiven), foldedLetters(p.seedFamily)
		for _, g := range p.givens {
			for _, f := range p.families {
				for gi, gf := range forms {
					for fi, ff := range forms {
						if !raceKeep(t, gi*10+fi, 3) {
							continue
						}
						in := "ping " + caseForm(g, gf) + " " + caseForm(f, ff) + " today"
						for _, mode := range []string{"same", "new", "reopen", "anchored"} {
							path := filepath.Join(t.TempDir(), "v.db")
							e, v := modeEngine(t, mode, path)
							s := e.NewSession("one")
							full := in
							if mode == "anchored" {
								full = seed + " " + in
							} else if _, err := s.Mask(ctx, seed); err != nil {
								t.Fatal(err)
							}
							switch mode {
							case "new":
								s = e.NewSession("two")
							case "reopen":
								v.Close()
								e, v = strictFileEngine(t, path)
								s = e.NewSession("two")
							}
							out, err := s.Mask(ctx, full)
							if err != nil {
								t.Fatal(err)
							}
							tail := out[strings.LastIndex(out, "ping "):]
							for _, tok := range strings.Fields(textutil.Fold(tail)) {
								if l := foldedLetters(tok); l == gk || l == fk {
									t.Errorf("%s: %q left %q raw in %q", mode, in, tok, out)
								}
							}
							if back := s.Rehydrate(out); back != full {
								t.Errorf("%s batch: %q restored as %q", mode, full, back)
							}
							if back := spellingStream(s, out); back != full {
								t.Errorf("%s stream: %q restored as %q", mode, full, back)
							}
							v.Close()
						}
					}
				}
			}
		}
	}
}

func TestMaskedVariantVoidsEveryVariantInText(t *testing.T) {
	for _, c := range []struct{ seed, in, name string }{
		{"Ruby Kelly called about the invoice.", "rUbY RuBy rUBY RUBy Ruby RUBY ruby called.", "ruby"},
		{"Zoë Kelly called about the invoice.", "Zoe ZOË zoë Zoë called.", "zoe"},
	} {
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			for _, tok := range strings.Fields(textutil.Fold(out)) {
				if foldedLetters(tok) == c.name {
					t.Errorf("%s: %q left a variant raw in %q", mode, c.in, out)
				}
			}
			if back != c.in {
				t.Errorf("%s: %q restored as %q", mode, c.in, back)
			}
		})
		s := newEngine(t, nil).NewSession("p")
		full := c.seed + " " + c.in
		out, _ := s.Mask(context.Background(), full)
		for _, tok := range strings.Fields(textutil.Fold(out)) {
			if foldedLetters(tok) == c.name {
				t.Errorf("anchored: %q left a variant raw in %q", full, out)
			}
		}
		if back := s.Rehydrate(out); back != full {
			t.Errorf("anchored: %q restored as %q", full, back)
		}
	}
}

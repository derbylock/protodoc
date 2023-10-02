package main

import "fmt"

func (pf ProtoFile) output() {
	pf.outputServices()
	pf.outputEnums()
	pf.outputObjects()
}

func (pf ProtoFile) outputServices() {
	for _, s := range pf.Services {

		// service line
		fmt.Printf("SERVICE %s\n\n", s.PackageName)

		// method group comment
		fmt.Printf("%s\n\n", s.Comment)

		// list of methods
		for _, inf := range s.Operations {

			// method line
			fmt.Printf("METHOD %s", inf.ServiceName+"."+inf.MethodName)
			if inf.Type != Unary {
				fmt.Printf(" (%s)\n", inf.Type)
			} else {
				fmt.Printf("\n")
			}

			// rest api line
			fmt.Printf("%s %s\n", inf.HTTPMethod, inf.URLPath)

			// method comments
			fmt.Printf("%s\n\n", inf.Comment)

			// method request
			fmt.Printf("REQUEST PARAMETERS (%s)\n", inf.Request.Type)
			for _, f := range inf.Request.Params {
				fmt.Printf("    %s %s %s\n", f.Type(), f.Name, f.Comment)
			}

			// method response
			fmt.Printf("RESPONSE PARAMETERS (%s)\n", inf.Response.Type)
			for _, f := range inf.Response.Params {
				fmt.Printf("    %s %s %s\n", f.Type(), f.Name, f.Comment)
			}

			// end of method
			fmt.Printf("\n")
		}
	}
}

func (pf ProtoFile) outputEnums() {
	for _, enum := range pf.Enums {
		fmt.Printf("ENUM %s\n", enum.Name)
		fmt.Printf("%s\n\n", enum.Comment)
		fmt.Printf("CONSTANTS\n")
		for _, c := range enum.Constants {
			fmt.Printf("    %s %s %s\n", c.Name, c.Val, c.Comment)
		}
		fmt.Printf("\n")
	}
}

func (pf ProtoFile) outputObjects() {
	for _, obj := range pf.Objects {
		fmt.Printf("OBJECT %s\n", obj.Name)
		fmt.Printf("%s\n\n", obj.Comment)
		fmt.Printf("ATTRIBUTES\n")
		for _, a := range obj.Attrs {
			fmt.Printf("    %s %s %s\n", a.Type(), a.Name, a.Comment)
		}
		fmt.Printf("\n")
	}
}

package main

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/common/prototype"
	"log"
)

func main() {
	//db, _ := storage.NewLevelDatabase("/Users/yykingking/abc123.db")
	db := storage.NewMemoryDatabase()

	defer db.Close()

	//1.create the table wrap
	//we can use the type  which is contained in another created pb struct,
	// such as "prototype.account_name" in AccountName 、prototype.time_point_sec
	//MakeXXX func can create a pb struct
	mKey := prototype.MakeAccountName("pbTool")
	wrap := table.NewSoDemoWrap(db, mKey)
	if wrap == nil {
		//crreate fail , the db already contain table with current mainKey
		log.Println("crreate fail , the db already contain table with current mainKey")
		return
	}

	//2.create the pb struct
	data := table.SoDemo{
	 	Owner:mKey,
	 	Title:"hello",
	 	Content:"test the pb tool",
	 	Idx: 1000,
	 	LikeCount:1,
	 	Taglist:"#NBA",
	 	ReplayCount:100,
	 }

	 //3.save table data to db
	 res := wrap.CreateDemo(&data)
	 if !res {
	 	 log.Fatalln("create new table of Demo faile")
		 return
	 }

	 /*
	   --------------------------
	   Get Property（the GetXXX function  return the point of value）
	   --------------------------*/

	 //get title
	 tPtr := wrap.GetTitle()
	 if tPtr != nil {
		 fmt.Printf("the title is %s \n",*tPtr)
	 }else {
		 fmt.Printf("get title fail")
	 }

	 //get content
	 cPtr := wrap.GetContent()
	if cPtr != nil {
		fmt.Printf("the content is %s \n",*cPtr)
	}else {
		log.Printf("modify tilte fail")
	}
	//modify title
	tMdRes := wrap.MdContent("hello world")
	if !tMdRes {
		log.Println("modify tilte fail")
	}


	/*
	  --------------------------
	   Modify property value (******can't modify the mainkey)
	  --------------------------*/
	//modify content
	cMdRes := wrap.MdContent("test md the content")
	if !cMdRes {
		log.Printf("modify content fail")
	}


	/*--------------------------
	   Sort Query List
	  --------------------------*/
     //1.create the sort wrap for property which is surpport sort (E.g postTime)
	 tSortWrap := table.SDemoPostTimeWrap{}
	tSortWrap.Dba = db
	 //2.start query data of range(if start graeater than sort by reverse order，otherwise sort by order)
	 iter := tSortWrap.QueryList(*prototype.MakeTimeSecondPoint(20136666),
	 	*prototype.MakeTimeSecondPoint(2013999))
	 //we can get the main key and sub key by the returned iterator
	 if iter != nil {
	 	for iter.Next() {
			//get the mainkey value (GetMainVal return the ptr of value)
			mKeyPtr := tSortWrap.GetMainVal(iter)
			if mKeyPtr == nil {
				fmt.Println("get main key fail")
			}
			//get subKey value (the postTime value)
			mSubPtr := tSortWrap.GetSubVal(iter)
			if mSubPtr == nil {
				fmt.Println("get postTime fail")
			}
		}

	 }else {
	 	log.Println("there is no data exist in range")
	 }

     //query single value but not a range,start and end set the same value
	iter1 := tSortWrap.QueryList(*prototype.MakeTimeSecondPoint(20136666),
		*prototype.MakeTimeSecondPoint(20136666))
	if iter1 != nil {
		mKeyPtr := tSortWrap.GetMainVal(iter1)
		if mKeyPtr == nil {
			fmt.Println("get main key fail")
		}
	}

	/*
	 --------------------------
	  unique Query List (only support query the property which is flag unique)
	 --------------------------*/
	 //1.create the uni wrap of property which is need unique query
	 var idx int64 = 100
	 uniWrap := table.UniDemoIdxWrap{}
	 //2.use UniQueryXX func to query data meanWhile return the table wrap
	  dWrap := uniWrap.UniQueryIdx(&idx)
	  t := dWrap.GetTitle()
	  fmt.Printf("the title of index is %s",*t)

	  //unique query mainkey(E.g query owner)
	   mUniWrap := table.UniDemoOwnerWrap{}
	   //
	   wrap1 := mUniWrap.UniQueryOwner(prototype.MakeAccountName("test"))
	   if wrap1 != nil {
	   	  fmt.Printf("owner is test,the idx is %s",*wrap1.GetIdx())
	   }

	  /*
	    remove tabale data from db
	  */
	  //judge the table of current mainKey if is exist
	  isExsit := wrap.CheckExist()
	  if isExsit {
	  	 res := wrap.RemoveDemo()
	  	 if !res {
	  	 	fmt.Println("remove the table data faile")
		 }
	  }
	
}
